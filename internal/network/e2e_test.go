package network

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdityaKrSingh26/PeerVault/internal/crypto"
	"github.com/AdityaKrSingh26/PeerVault/internal/storage"
	"github.com/AdityaKrSingh26/PeerVault/pkg/p2p"
	"github.com/stretchr/testify/assert"
)

func TestE2EReplicationAndRetrieval(t *testing.T) {
	// Setup temporary storage roots
	root1 := filepath.Join(os.TempDir(), "pv_e2e_node1")
	root2 := filepath.Join(os.TempDir(), "pv_e2e_node2")
	os.RemoveAll(root1)
	os.RemoveAll(root2)
	defer os.RemoveAll(root1)
	defer os.RemoveAll(root2)

	encKey, _ := crypto.NewEncryptionKey()
	id1, err := crypto.GenerateID()
	assert.Nil(t, err)
	id2, err := crypto.GenerateID()
	assert.Nil(t, err)

	// Start Node 1 (port 5000)
	opts1 := FileServerOpts{
		StorageRoot:       root1,
		PathTransformFunc: storage.CASPathTransformFunc,
		ID:                id1,
		EncKey:            encKey,
	}
	server1 := NewFileServer(opts1)
	tr1 := p2p.NewTCPTransport(p2p.TCPTransportOpts{
		ListenAddr:    ":5000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	})
	tr1.OnPeer = server1.OnPeer
	server1.Transport = tr1
	server1.Pex = NewPeerExchangeService(server1)

	// Start Node 2 (port 6000)
	opts2 := FileServerOpts{
		StorageRoot:       root2,
		PathTransformFunc: storage.CASPathTransformFunc,
		ID:                id2,
		EncKey:            encKey,
	}
	server2 := NewFileServer(opts2)
	tr2 := p2p.NewTCPTransport(p2p.TCPTransportOpts{
		ListenAddr:    ":6000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	})
	tr2.OnPeer = server2.OnPeer
	server2.Transport = tr2
	server2.Pex = NewPeerExchangeService(server2)

	// Start servers
	go server1.Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	go server2.Start(context.Background())
	time.Sleep(100 * time.Millisecond)

	defer server1.Stop()
	defer server2.Stop()

	// Connect Node 2 to Node 1
	err = server2.Transport.Dial("127.0.0.1:5000")
	assert.Nil(t, err)

	// Connect Node 1 to Node 2 (make connection bi-directional)
	err = server1.Transport.Dial("127.0.0.1:6000")
	assert.Nil(t, err)

	// Give time to exchange peers/handshake
	time.Sleep(200 * time.Millisecond)

	// Store file on Node 1
	fileKey := "secret_doc.txt"
	fileContent := []byte("PeerVault E2E test file content - Top Secret!")
	err = server1.Store(context.Background(), fileKey, bytes.NewReader(fileContent))
	assert.Nil(t, err)

	// Give time to replicate to Node 2
	time.Sleep(200 * time.Millisecond)

	// Retrieve file from Node 2 (since Node 2 is a peer, it should already have it replicated)
	reader, err := server2.Get(context.Background(), fileKey)
	assert.Nil(t, err)

	retrievedContent, err := io.ReadAll(reader)
	assert.Nil(t, err)
	assert.Equal(t, string(fileContent), string(retrievedContent))
}
