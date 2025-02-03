package p2p

import "net"

// Peer is an interface that represents the remote node.
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport is anything that handles the communication
// between the nodes in the network. This can be of the
// form (TCP, UDP, websockets, ...)
type Transport interface {
	Addr() string           //Return the address it’s listening on
	Dial(string) error      //Connect to another node at a specific address.
	ListenAndAccept() error //Start listening for incoming connections from other nodes.
	Consume() <-chan RPC    //channel that receives incoming RPC messages from other nodes.
	Close() error           //shut down the transport and clean up the resources.
}
