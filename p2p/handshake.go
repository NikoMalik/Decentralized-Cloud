package p2p

// handshake func is
type HandshakeFunc func(Peer) error

func NopHandshakeFunc(Peer) error {
	return nil
}
