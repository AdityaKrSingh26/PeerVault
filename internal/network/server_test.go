package network

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPEXGOBRegistration(t *testing.T) {
	// Verify MessagePeerExchange GOB encoding and decoding works
	pexMsg := MessagePeerExchange{
		Peers: []PeerInfo{
			{
				Address:  "127.0.0.1:4000",
				LastSeen: time.Now(),
				Source:   "pex",
			},
		},
	}

	msg := Message{
		Payload: pexMsg,
	}

	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(&msg)
	assert.Nil(t, err)

	var decodedMsg Message
	err = gob.NewDecoder(buf).Decode(&decodedMsg)
	assert.Nil(t, err)

	decodedPex, ok := decodedMsg.Payload.(MessagePeerExchange)
	assert.True(t, ok)
	assert.Equal(t, 1, len(decodedPex.Peers))
	assert.Equal(t, "127.0.0.1:4000", decodedPex.Peers[0].Address)
	assert.Equal(t, "pex", decodedPex.Peers[0].Source)
}

func TestStreamHeaderGOBRegistration(t *testing.T) {
	// Verify StreamHeader GOB encoding and decoding works
	header := StreamHeader{
		ID:   "node-1",
		Key:  "test-key",
		Size: 1024,
	}

	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(&header)
	assert.Nil(t, err)

	var decodedHeader StreamHeader
	err = gob.NewDecoder(buf).Decode(&decodedHeader)
	assert.Nil(t, err)

	assert.Equal(t, "node-1", decodedHeader.ID)
	assert.Equal(t, "test-key", decodedHeader.Key)
	assert.Equal(t, int64(1024), decodedHeader.Size)
}
