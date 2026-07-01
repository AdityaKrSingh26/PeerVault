package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AdityaKrSingh26/PeerVault/internal/crypto"
)

const defaultRootFolderName = "storage/default"

type PathKey struct {
	PathName string // The directory structure where the file will be stored
	Filename string // The actual filename
}

// Defines a function type that transforms a key into a PathKey
type PathTransformFunc func(string) PathKey

// defines configuration options for the storage system
type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts                   // Embeds StoreOpts (inherits its fields)
	keyMap    map[string]string // Maps hash -> original key
	keyMapMu  sync.RWMutex      // Protects keyMap access
}

// Generates a unique directory structure and filename for a given key using a SHA-256 hash.
func CASPathTransformFunc(key string) PathKey {
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// Splits the hash string into chunks of 5 characters each.
	blocksize := 5
	sliceLen := len(hashStr) / blocksize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

// PathKey method to get the first directory from the full path
func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0] // Return the first part
}

// PathKey method to get the full path (folder structure + filename)
func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

// Default path transformation function (uses the key directly)
var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

// NewStore initializes a new Store with given options
func NewStore(opts StoreOpts) *Store {

	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	s := &Store{
		StoreOpts: opts,
		keyMap:    make(map[string]string),
	}

	// Load keys if they exist on disk
	_ = s.loadKeyMap()

	return s
}

func ValidateNodeID(id string) error {
	if id == "" {
		return errors.New("nodeID cannot be empty")
	}
	if strings.ContainsAny(id, "/\\.") {
		return fmt.Errorf("nodeID contains invalid characters: %q", id)
	}
	// Must be hex string (SHA-256 IDs are hex)
	if _, err := hex.DecodeString(id); err != nil {
		return fmt.Errorf("nodeID must be hex: %w", err)
	}
	return nil
}

func (s *Store) resolvePath(id string, subpath string) (string, error) {
	if err := ValidateNodeID(id); err != nil {
		return "", err
	}
	cleanRoot := filepath.Clean(s.Root)
	resolved := filepath.Join(cleanRoot, id, subpath)

	// Ensure resolved path is under cleanRoot
	prefix := cleanRoot + string(os.PathSeparator)
	if !strings.HasPrefix(resolved, prefix) && resolved != cleanRoot {
		return "", fmt.Errorf("path escape detected: %s", resolved)
	}
	return resolved, nil
}

// checks if a file exists in the store
func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot, err := s.resolvePath(id, pathKey.FullPath())
	if err != nil {
		return false
	}

	_, err = os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

// Clear deletes the entire storage root folder and its contents
func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// Delete removes a specific file and its associated directories
func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()

	firstPathNameWithRoot, err := s.resolvePath(id, pathKey.FirstPathName())
	if err != nil {
		return err
	}

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	// Store the key mapping
	pathKey := s.PathTransformFunc(key)

	s.keyMapMu.Lock()
	s.keyMap[pathKey.Filename] = key
	s.keyMapMu.Unlock()

	_ = s.saveKeyMap()

	return s.writeStream(id, key, r)
}

// writes encrypted data to a file
func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := crypto.CopyDecrypt(encKey, r, f)

	return int64(n), err
}

// writes encrypted data to a file (encrypting on-the-fly)
func (s *Store) WriteEncrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	// Store the key mapping
	pathKey := s.PathTransformFunc(key)

	s.keyMapMu.Lock()
	s.keyMap[pathKey.Filename] = key
	s.keyMapMu.Unlock()

	_ = s.saveKeyMap()

	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := crypto.CopyEncrypt(encKey, r, f)

	return int64(n), err
}

// openFileForWriting ensures the necessary directories exist and opens the file
func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot, err := s.resolvePath(id, pathKey.PathName)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot, err := s.resolvePath(id, pathKey.FullPath())
	if err != nil {
		return nil, err
	}

	return os.Create(fullPathWithRoot)
}

// writes data from an io.Reader to the file
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return io.Copy(f, r)
}

func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

// readStream opens a file and returns its reader
func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot, err := s.resolvePath(id, pathKey.FullPath())
	if err != nil {
		return 0, nil, err
	}

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	// Return file size and reader
	return fileInfo.Size(), file, nil
}

// FileInfo represents information about a stored file
type FileInfo struct {
	Key    string // Original file key
	Hash   string // File hash (filename)
	Size   int64  // File size in bytes
	NodeID string // ID of the node that stored it
}

// List returns information about all files stored for a given node ID
func (s *Store) List(id string) ([]FileInfo, error) {
	var files []FileInfo

	nodeDir, err := s.resolvePath(id, "")
	if err != nil {
		return nil, err
	}

	// Check if node directory exists
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return files, nil // Return empty list if no files stored yet
	}

	// Walk through all files in the node's directory
	err = filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, only process files
		if info.IsDir() {
			return nil
		}

		// The filename is the hash, we need to find the original key
		hash := info.Name()

		// Try to get the original key from our mapping
		s.keyMapMu.RLock()
		originalKey, exists := s.keyMap[hash]
		s.keyMapMu.RUnlock()

		if !exists {
			// If not in mapping, use abbreviated hash as display name
			originalKey = fmt.Sprintf("file_%s", hash[:8])
		}

		fileInfo := FileInfo{
			Key:    originalKey,
			Hash:   hash,
			Size:   info.Size(),
			NodeID: id,
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

// ListAll returns information about all files stored across all nodes
func (s *Store) ListAll() (map[string][]FileInfo, error) {
	allFiles := make(map[string][]FileInfo)

	// Check if root directory exists
	if _, err := os.Stat(s.Root); os.IsNotExist(err) {
		return allFiles, nil
	}

	// Read all node directories
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return allFiles, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nodeID := entry.Name()
			files, err := s.List(nodeID)
			if err != nil {
				continue // Skip problematic directories
			}
			if len(files) > 0 {
				allFiles[nodeID] = files
			}
		}
	}

	return allFiles, nil
}

// GetOriginalKey returns the original key for a given hash.
func (s *Store) GetOriginalKey(hash string) (string, bool) {
	s.keyMapMu.RLock()
	defer s.keyMapMu.RUnlock()
	key, exists := s.keyMap[hash]
	return key, exists
}

// ClearKeyMap safely clears the key mapping
func (s *Store) ClearKeyMap() {
	s.keyMapMu.Lock()
	s.keyMap = make(map[string]string)
	s.keyMapMu.Unlock()

	_ = s.saveKeyMap()
}

func (s *Store) saveKeyMap() error {
	s.keyMapMu.RLock()
	defer s.keyMapMu.RUnlock()

	metadataPath := filepath.Join(s.Root, "metadata.json")
	if err := os.MkdirAll(s.Root, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.keyMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

func (s *Store) loadKeyMap() error {
	metadataPath := filepath.Join(s.Root, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}

	s.keyMapMu.Lock()
	defer s.keyMapMu.Unlock()
	return json.Unmarshal(data, &s.keyMap)
}
