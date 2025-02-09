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
	net.Conn                 
	outbound bool            
	wg       *sync.WaitGroup 
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
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// send data to remote node
func (p *TCPPeer) Send(B []byte) error {
	_, err := p.Conn.Write(B)
	return err
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

// manage TCP connections and communication with other nodes.
type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

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

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// close TCP listner and stop receiving new connections
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

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

	defer func() {
		fmt.Printf("Dropping peer connections: %s", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}
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
