package p2p

import (
	"encoding/gob"
	"io"
)

type DefaultDecoder struct{}

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

// Decode reads data from the stream and processes it based on the first byte
func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	// Create a 1-byte buffer to peek at the first byte of the stream
	peekBuf := make([]byte, 1)

	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// Handle stream flag
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	// Handle regular payload
	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]
	return nil
}
