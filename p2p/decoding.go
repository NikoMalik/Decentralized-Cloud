package p2p

import (
	"bufio"
	"encoding/gob"
	"io"
	"sync"

	"github.com/NikoMalik/Decentralized-Cloud/buffer"
)

type Decoder interface {
	Decode(io.Reader, *Message) error

	Encode(*bufio.Writer, any) error
}

var bufioWriterPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReaderSize(nil, 8)
	},
}

type P2PDecoder struct {
	buffer buffer.RingBuffer
}

type DefaulDecoder struct {
}

func (d *P2PDecoder) Encode(w *bufio.Writer, v any) error {
	return gob.NewEncoder(w).Encode(v)
}

// transmit and decode structured data
func (dec *P2PDecoder) Decode(r io.Reader, msg *Message) error {

	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// In case of a stream we are not decoding what is being sent over the network.
	// We are just setting Stream true so we can handle that in our logic.
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]

	return nil
}

// just simple binary transfer

type Writer struct {
	W    io.Writer
	bufw *bufio.Writer
}

// Write writes the given byte slice to the underlying connection.
//
// Note: Write won't return the write buffer to the pool even if it ends up
// being empty after the write. You must call Flush() to do that.
func (w *Writer) Write(b []byte) (int, error) {

	if w.bufw == nil {
		if len(b) >= 1024 {
			return w.W.Write(b)
		}
		w.bufw = bufioWriterPool.Get().(*bufio.Writer)
		w.bufw.Reset(w.W)
	}

	return w.bufw.Write(b)

}

func (w *Writer) ensureBuffer() error {
	if w.bufw == nil {
		w.bufw = bufioWriterPool.Get().(*bufio.Writer)
		w.bufw.Reset(w.W)
	}
	return nil
}

func (w *Writer) Size() int {
	WriterBufferSize := 1024
	return WriterBufferSize

}

func (w *Writer) Available() int {
	if w.bufw != nil {
		return w.bufw.Available()
	}
	return 1024
}

func (w *Writer) Buffered() int {
	if w.bufw != nil {
		return w.bufw.Buffered()
	}
	return 0
}

func (w *Writer) WriteRune(r rune) (int, error) {

	w.ensureBuffer()
	return w.bufw.WriteRune(r)
}

// Flush flushes the write buffer, if any, and returns it to the pool.
func (w *Writer) Flush() error {
	if w.bufw == nil {
		return nil
	}
	if err := w.bufw.Flush(); err != nil {
		return err
	}
	w.bufw.Reset(nil)
	bufioWriterPool.Put(w.bufw)
	w.bufw = nil
	return nil
}

func (w *Writer) Close() error {
	var (
		ferr, cerr error
	)
	ferr = w.Flush()

	// always close even if flush fails.
	if closer, ok := w.W.(io.Closer); ok {
		cerr = closer.Close()
	}

	if ferr != nil {
		return ferr
	}
	return cerr
}
