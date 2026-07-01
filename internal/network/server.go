package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/AdityaKrSingh26/PeerVault/internal/crypto"
	"github.com/AdityaKrSingh26/PeerVault/internal/metrics"
	"github.com/AdityaKrSingh26/PeerVault/internal/quota"
	"github.com/AdityaKrSingh26/PeerVault/internal/storage"
	"github.com/AdityaKrSingh26/PeerVault/pkg/p2p"
)

// configuration options
type FileServerOpts struct {
	ID                string
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc storage.PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

// StreamHeader represents the header of a file stream sent over the network.
type StreamHeader struct {
	ID   string
	Key  string
	Size int64
}

// Manages file storage, peer connections, and network communication.
type FileServer struct {
	FileServerOpts

	PeerLock sync.Mutex
	Peers    map[string]p2p.Peer

	store        *storage.Store
	QuotaManager *quota.QuotaManager
	GC           *storage.GarbageCollector
	Metrics      *metrics.Metrics
	Discovery    *DiscoveryService
	Pex          *PeerExchangeService
	quitch       chan struct{}

	waitersMu sync.Mutex
	waiters   map[string][]chan struct{}
}

// Initializes a new "FileServer" instance.
func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := storage.StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	if len(opts.ID) == 0 {
		id, err := crypto.GenerateID()
		if err != nil {
			log.Fatalf("failed to generate secure node ID: %v", err)
		}
		opts.ID = id
	}

	if err := storage.ValidateNodeID(opts.ID); err != nil {
		log.Fatalf("invalid node ID %q: %v", opts.ID, err)
	}

	store := storage.NewStore(storeOpts)
	quotaManager := quota.NewQuotaManager(opts.StorageRoot)
	gc := storage.NewGarbageCollector(store, opts.ID)
	metricsObj := metrics.NewMetrics()

	server := &FileServer{
		FileServerOpts: opts,
		store:          store,
		QuotaManager:   quotaManager,
		GC:             gc,
		Metrics:        metricsObj,
		quitch:         make(chan struct{}),
		Peers:          make(map[string]p2p.Peer),
		waiters:        make(map[string][]chan struct{}),
	}

	server.Pex = NewPeerExchangeService(server)
	return server
}

// Sends a message to all connected peers.
func (s *FileServer) broadcast(msg *Message) error {
	s.PeerLock.Lock()
	defer s.PeerLock.Unlock()

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	var failed []string
	for addr, peer := range s.Peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			failed = append(failed, addr)
			log.Printf("WARN broadcast failed to peer %s: %v", addr, err)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("broadcast failed to %d peer(s): %v", len(failed), failed)
	}
	return nil
}

// Generic message wrapper
type Message struct {
	Payload any
}

// Notifies peers about a file being stored
type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

// Requests a file from peers
type MessageGetFile struct {
	ID  string
	Key string
}

// decryptOnTheFly decrypts an encrypted reader stream on-the-fly using io.Pipe
func (s *FileServer) decryptOnTheFly(ctx context.Context, r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer func() {
			if rc, ok := r.(io.Closer); ok {
				rc.Close()
			}
		}()

		errChan := make(chan error, 1)
		go func() {
			_, err := crypto.CopyDecrypt(s.EncKey, r, pw)
			errChan <- err
		}()

		select {
		case err := <-errChan:
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		case <-ctx.Done():
			pw.CloseWithError(ctx.Err())
		}
	}()
	return pr
}

// Retrieves a file from the local store or fetches it from the network.
func (s *FileServer) Get(ctx context.Context, key string) (io.Reader, error) {

	// Checks if the file exists locally.
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(s.ID, key)
		if err != nil {
			return nil, err
		}
		return s.decryptOnTheFly(ctx, r), nil
	}

	fmt.Printf("[%s] dont have file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

	ch := s.registerFileWaiter(key)

	// If not, broadcasts a MessageGetFile request to peers.
	msg := Message{
		Payload: MessageGetFile{
			ID:  s.ID,
			Key: crypto.HashKey(key),
		},
	}
	if err := s.broadcast(&msg); err != nil {
		log.Printf("Warning: file request broadcast encountered errors: %v", err)
	}

	// Wait for notification or timeout
	select {
	case <-ch:
		// File was successfully received and written to disk
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("file %s not found on the network (timeout)", key)
	}

	_, r, err := s.store.Read(s.ID, key)
	if err != nil {
		return nil, err
	}
	return s.decryptOnTheFly(ctx, r), nil
}

// Stores a file locally and notifies peers.
func (s *FileServer) Store(ctx context.Context, key string, r io.Reader) error {
	// Store encrypted locally (streaming / constant memory)
	size, err := s.store.WriteEncrypt(s.EncKey, s.ID, key, r)
	if err != nil {
		return err
	}

	s.PeerLock.Lock()
	defer s.PeerLock.Unlock()

	// Stream to all connected peers concurrently
	for _, peer := range s.Peers {
		go func(p p2p.Peer) {
			if ctx.Err() != nil {
				return
			}
			_, fileReader, err := s.store.Read(s.ID, key)
			if err != nil {
				log.Printf("failed to read local file for streaming: %v", err)
				return
			}
			defer func() {
				if closer, ok := fileReader.(io.Closer); ok {
					closer.Close()
				}
			}()

			if err := s.sendStream(p, key, size, fileReader); err != nil {
				log.Printf("failed to send stream to peer: %v", err)
			}
		}(peer)
	}

	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

// Handles new peer connections.
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.PeerLock.Lock()
	defer s.PeerLock.Unlock()

	// Adds the peer to the peers map.
	s.Peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr())

	return nil
}

func (s *FileServer) registerFileWaiter(key string) chan struct{} {
	s.waitersMu.Lock()
	defer s.waitersMu.Unlock()

	ch := make(chan struct{}, 1)
	hashedKey := crypto.HashKey(key)
	s.waiters[hashedKey] = append(s.waiters[hashedKey], ch)
	return ch
}

func (s *FileServer) notifyFileWaiter(hashedKey string) {
	s.waitersMu.Lock()
	defer s.waitersMu.Unlock()

	channels, exists := s.waiters[hashedKey]
	if !exists {
		return
	}

	for _, ch := range channels {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	delete(s.waiters, hashedKey)
}

func (s *FileServer) sendStream(peer p2p.Peer, key string, size int64, r io.Reader) error {
	if err := peer.Send([]byte{p2p.IncomingStream}); err != nil {
		return err
	}

	header := StreamHeader{
		ID:   s.ID,
		Key:  key,
		Size: size,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(&header); err != nil {
		return err
	}

	headerSize := int16(buf.Len())
	if err := binary.Write(peer, binary.LittleEndian, headerSize); err != nil {
		return err
	}
	if err := peer.Send(buf.Bytes()); err != nil {
		return err
	}

	_, err := io.Copy(peer, r)
	return err
}

func (s *FileServer) handleStream(from string) error {
	s.PeerLock.Lock()
	peer, ok := s.Peers[from]
	s.PeerLock.Unlock()
	if !ok {
		return fmt.Errorf("peer %s not found in map", from)
	}

	defer peer.CloseStream()

	var headerSize int16
	if err := binary.Read(peer, binary.LittleEndian, &headerSize); err != nil {
		return err
	}

	headerBuf := make([]byte, headerSize)
	if _, err := io.ReadFull(peer, headerBuf); err != nil {
		return err
	}

	var header StreamHeader
	if err := gob.NewDecoder(bytes.NewReader(headerBuf)).Decode(&header); err != nil {
		return err
	}

	_, err := s.store.Write(s.ID, header.Key, io.LimitReader(peer, header.Size))
	if err != nil {
		return err
	}

	s.notifyFileWaiter(header.Key)

	return nil
}

// Main event loop for handling incoming messages.
func (s *FileServer) loop(ctx context.Context) {
	defer func() {
		log.Println("file server stopped due to error or user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			if rpc.Stream {
				if err := s.handleStream(rpc.From); err != nil {
					log.Println("handle stream error: ", err)
				}
				continue
			}

			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}
			if err := s.handleMessage(ctx, rpc.From, &msg); err != nil {
				log.Println("handle message error: ", err)
			}

		case <-s.quitch:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Processes incoming messages.
func (s *FileServer) handleMessage(ctx context.Context, from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	case MessagePeerExchange:
		return s.handleMessagePeerExchange(ctx, from, v)
	}

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	originalKey, exists := s.store.GetOriginalKey(msg.Key)
	if !exists || !s.store.Has(s.ID, originalKey) {
		return fmt.Errorf("[%s] need to serve file (%s) but it does not exist on disk", s.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", s.Transport.Addr(), originalKey)

	fileSize, r, err := s.store.Read(s.ID, originalKey)
	if err != nil {
		return err
	}
	defer r.(io.Closer).Close()

	peer, ok := s.Peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	return s.sendStream(peer, originalKey, fileSize, r)
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s] attemping to connect with remote %s\n", s.Transport.Addr(), addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (s *FileServer) Start(ctx context.Context) error {
	fmt.Printf("[%s] starting fileserver...\n", s.Transport.Addr())

	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNetwork()

	if s.GC != nil {
		s.GC.Start(ctx)
	}

	s.loop(ctx)

	return nil
}

func init() {
	gob.Register(MessageGetFile{})
	gob.Register(StreamHeader{})
	gob.Register(MessagePeerExchange{})
	gob.Register(PeerInfo{})
}

// Delete removes a file from local storage and broadcasts deletion to peers

// Delete removes a file
func (s *FileServer) Delete(key string) error {
	if !s.store.Has(s.ID, key) {
		return fmt.Errorf("file not found")
	}
	return s.store.Delete(s.ID, key)
}

// EnableLocalDiscovery enables mDNS discovery
func (s *FileServer) EnableLocalDiscovery(ctx context.Context, advertiseAddr string) error {
	s.Discovery = NewDiscoveryService("peervault", 3000, advertiseAddr)
	s.Discovery.SetPeerFoundCallback(func(peerAddr string) error {
		return s.Transport.Dial(peerAddr)
	})
	return s.Discovery.Start(ctx)
}

// EnablePeerExchange enables PEX
func (s *FileServer) EnablePeerExchange(ctx context.Context) {
	if s.Pex != nil {
		s.Pex.Start(ctx)
	}
}

// Public store accessors
func (s *FileServer) ListFiles(id string) ([]storage.FileInfo, error) {
	return s.store.List(id)
}

func (s *FileServer) ListAllFiles() (map[string][]storage.FileInfo, error) {
	return s.store.ListAll()
}

func (s *FileServer) ReadFile(id, key string) (int64, io.Reader, error) {
	return s.store.Read(id, key)
}

func (s *FileServer) ClearStorage() error {
	return s.store.Clear()
}

func (s *FileServer) ClearKeyMapping() {
	s.store.ClearKeyMap()
}
