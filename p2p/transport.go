package p2p

import "net"

// peer represents the remote node
type Peer interface {
}

// transport is handlers the communication
// between the nodes  in the network

// form (tcp, udp, websockets)
type Transport interface {
	ListenAndAccept() error
	Connections() <-chan net.Conn
	GetActiveConnections() int32
	GetPeers() []Peer
	Stop() error
}
