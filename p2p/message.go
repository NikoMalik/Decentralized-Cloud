package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// just for making sense
type Message struct {
	Payload []byte
	Type    int
	From    string
	Stream  bool
	Size    int
}
