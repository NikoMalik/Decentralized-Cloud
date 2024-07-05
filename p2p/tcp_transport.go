package p2p

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/NikoMalik/Decentralized-Cloud/stack"
	"github.com/fatih/color"
)

// TcpTransport represents the TCP transport layer.
// TcpTransport represents the TCP transport layer.
type TcpTransport struct {
	ListenAddr string            // ip:port
	Listener   net.Listener      // get all incoming tcp connections
	Peers      map[net.Addr]Peer // map of all connected peers
	ActiveConn int32             // number of active connections for safety changes in concurrency space
	mu         sync.RWMutex      // safety access for map Peers
	connChan   chan net.Conn     // channel for transmitting received connections

	messageChan chan Message

	closed atomic.Bool

	handshake HandshakeFunc
	wg        sync.WaitGroup
	Decoder   Decoder
	// RingBuffer *buffer.RingBuffer // RingBuffer instance

	ConnStack *stack.Stack[net.Conn] // using lock-free to manage connections

}

// TcpPeer represents a remote node over a tcp connection.
type TcpPeer struct {
	conn     net.Conn
	outbound bool // if dial to peer  == true, if we accept  == false
}

// NewTcpTransport creates a new TCP transport.
func NewTcpTransport(listenAddr string) (*TcpTransport, error) {
	// rb := buffer.NewMagicBuffer(buffer.DefaultMagicBufferSize)
	// if rb == nil {
	// 	return nil, errors.New("failed to create RingBuffer")
	// }
	return &TcpTransport{
		ListenAddr:  listenAddr,
		Peers:       make(map[net.Addr]Peer),
		connChan:    make(chan net.Conn),
		handshake:   NopHandshakeFunc,
		Decoder:     &P2PDecoder{},
		messageChan: make(chan Message),
		// RingBuffer:  rb,

		ConnStack: stack.NewLockFreeStack[net.Conn](),
	}, nil
}

// NewTcpPeer creates a new TCP peer.
func NewTcpPeer(conn net.Conn, outbound bool) *TcpPeer {
	return &TcpPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// ListenAndAccept starts listening and accepting connections.
func (t *TcpTransport) ListenAndAccept() error {
	var err error
	t.Listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.acceptConnections()
	}()
	return nil
}

// Connections returns the connection channel.
func (t *TcpTransport) Connections() <-chan net.Conn {
	return t.connChan
}

func (t *TcpTransport) Messages() <-chan Message {
	return t.messageChan
}

// GetActiveConnections returns the number of active connections.
func (t *TcpTransport) GetActiveConnections() int32 {
	connCount := atomic.LoadInt32(&t.ActiveConn)
	fmt.Printf("Active connections: %d\n", connCount)
	return connCount
}

// GetPeers returns a list of peers.
func (t *TcpTransport) GetPeers() []Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	peers := make([]Peer, 0, len(t.Peers))
	for _, peer := range t.Peers {
		peers = append(peers, peer)
	}
	return peers
}

// Stop stops the transport.
// Stop stops the transport by closing the listener and waiting for all goroutines to finish.
func (t *TcpTransport) Stop() error {
	if err := t.Listener.Close(); err != nil {
		return err
	}

	// Wait for all goroutines to finish.
	t.wg.Wait()

	// Release the RingBuffer.

	return nil
}

func (t *TcpTransport) RemovePeer(addr net.Addr) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.Peers, addr)
}

func (t *TcpTransport) HandleMessage(msg *Message) error {
	source := color.CyanString(msg.From)
	if msg.Payload == nil {
		return errors.New("payload is nil")
	}

	payloadStr := *(*string)(unsafe.Pointer(&msg.Payload))
	color.HiBlack("Message received from %s: %s", source, payloadStr)
	// Handle the message (this is a placeholder for actual message processing logic)
	return nil

}

// HandleConnection handles an incoming connection.
func (t *TcpTransport) HandleConnection() {
	conn := t.ConnStack.Pop()
	if conn == nil {
		color.Red("failed to pop connection from the stack")
		return
	}

	color.Cyan("new connection: %s", conn.RemoteAddr().String())

	peer := NewTcpPeer(conn, true)
	if err := t.handshake(peer); err != nil {
		color.Red("handshake error with peer %s: %s", conn.RemoteAddr().String(), err)
		conn.Close()
		return
	}

	t.mu.Lock()
	t.Peers[conn.RemoteAddr()] = peer
	t.mu.Unlock()

	defer conn.Close()

	for {
		var msg Message

		if err := t.Decoder.Decode(conn, &msg); err != nil {
			if err == io.EOF {
				color.Blue("connection with peer %s closed", conn.RemoteAddr().String())
				atomic.AddInt32(&t.ActiveConn, -1)
				t.GetActiveConnections()
				break
			} else {
				color.Red("error decoding message from peer %s: %s", conn.RemoteAddr().String(), err)
				atomic.AddInt32(&t.ActiveConn, -1)
				t.GetActiveConnections()
				break
			}
		}

		msg.From = conn.RemoteAddr().String()

		if msg.Stream {
			t.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			t.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}

		t.messageChan <- msg
	}
	t.mu.Lock()
	delete(t.Peers, conn.RemoteAddr())
	t.mu.Unlock()
}

// acceptConnections accepts incoming connections.
func (t *TcpTransport) acceptConnections() {
	for {
		conn, err := t.Listener.Accept()
		if err != nil {
			color.Magenta("error accepting connection:", err)
			return
		}

		atomic.AddInt32(&t.ActiveConn, 1)
		t.connChan <- conn

		t.ConnStack.Push(conn)

		t.wg.Add(1)
		go func(conn net.Conn) {
			defer t.wg.Done()
			t.HandleConnection()
		}(conn)
	}
}
