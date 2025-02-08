package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer is a struct that implements the Peer interface and represents a connection to another node over TCP.
type TCPPeer struct {
	net.Conn                 //underlying TCP connection
	outbound bool            //Indicates whether this connection was created by dialing out (true) or by accepting an incoming connection (false)
	wg       *sync.WaitGroup //manage synchronization when handling streams of data.
}

// Creates a new TCPPeer instance.
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// Signals that a stream of data has finished.
// (p *TCPPeer) is the method receiver.
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// send data to remote node
func (p *TCPPeer) Send(B []byte) error {
	_, err := p.Conn.Write(B)
	return err
}

// configuration option for TCPTansport
type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc    //perform a handshake when on new connection
	Decoder       Decoder          //decode incoming messages
	OnPeer        func(Peer) error //callback when a new peer is connected
}

// manage TCP connections and communication with other nodes.
type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener //accepting incoming connections.
	rpcch    chan RPC     //channel for receiving incoming RPC (Remote Procedure Call) messages.
}

// Creates a new TCPTransport instance.
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC, 1024),
	}
}

// Return the address it’s listening on
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// read only channel that receives incoming RPC messages.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// close TCP listner and stop receiving new connections
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// dial a remote node and establish a connection.
// implements the Transport interface.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil
}

// start listening for incoming connections.
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()
	log.Printf("TCP transport listening on %s\n", t.ListenAddr)
	return nil
}

// accept incoming connections and handle them in a sperate goroutine.
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Printf("TCP Error accepting connection: %s\n", err)
		}
		go t.handleConn(conn, false)
	}
}

// handle incoming connections and perform handshake.
// steps :
// 1. Creates a TCPPeer for the connection.
// 2. Performs a handshake.
// 3. Calls the OnPeer callback. Notifies the application that a new peer has been connected.
// 4. Enters a read loop to decode and process incoming messages.
// 5. If the message is a stream, it waits for the stream to finish before continuing.
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	// Ensures the connection is closed when the function exits, even if an error occurs.
	defer func() {
		fmt.Printf("Dropping peer connections: %s", err)
		conn.Close()
	}()

	// 1. Creates a TCPPeer for the connection.
	peer := NewTCPPeer(conn, outbound)

	// 2. Performs a handshake to verify the connection.
	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	// 3. Calls the OnPeer callback. Notifies the application that a new peer has been connected.
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// 4. Enters a read loop to decode and process incoming messages.
	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}
		// Set the Source of the Message:
		rpc.From = conn.RemoteAddr().String()
		// If the message is a stream, it waits for the stream to finish.
		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}
		t.rpcch <- rpc
	}
}
