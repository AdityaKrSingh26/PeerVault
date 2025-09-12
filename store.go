package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
}

// Generates a unique directory structure and filename for a given key using a SHA-1 hash.
func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
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

	return &Store{
		StoreOpts: opts,
		keyMap:    make(map[string]string),
	}
}

// checks if a file exists in the store
func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
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

	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	// Store the key mapping
	pathKey := s.PathTransformFunc(key)
	s.keyMap[pathKey.Filename] = key

	return s.writeStream(id, key, r)
}

// writes encrypted data to a file
func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encKey, r, f)

	return int64(n), err
}

// openFileForWriting ensures the necessary directories exist and opens the file
func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	return os.Create(fullPathWithRoot)
}

// writes data from an io.Reader to the file
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	return io.Copy(f, r)
}

func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

// readStream opens a file and returns its reader
func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

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

	nodeDir := fmt.Sprintf("%s/%s", s.Root, id)

	// Check if node directory exists
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return files, nil // Return empty list if no files stored yet
	}

	// Walk through all files in the node's directory
	err := filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
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
		originalKey, exists := s.keyMap[hash]
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
