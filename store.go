package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

type PathKey struct {
	PathName string // The directory structure where the file will be stored
	Filename string // The actual filename
}

// Defines a function type that transforms a key into a PathKey
type PathTransformFunc func(string) PathKey

// defines configuration options for the storage system
type StoreOpts struct {
	Root              string            // The root directory where files are stored
	PathTransformFunc PathTransformFunc // Function to transform keys into paths
}

type Store struct {
	StoreOpts // Embeds StoreOpts (inherits its fields)
}

// Generates a unique directory structure and filename for a given key using a SHA-1 hash.
func CASPathTransformFunc(key string) PathKey {
	// Computes the SHA-1 hash of the key.
	hash := sha1.Sum([]byte(key))
	// Encodes the hash as a hexadecimal string.
	hashStr := hex.EncodeToString(hash[:])

	// Splits the hash string into chunks of 5 characters each.
	blocksize := 5
	// Number of folders we will create
	sliceLen := len(hashStr) / blocksize
	// Slice to store folder names
	paths := make([]string, sliceLen)

	// Split the hash into equal-sized folder names
	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashStr[from:to]
	}

	// Return the generated PathKey
	return PathKey{
		PathName: strings.Join(paths, "/"), // Join the folders with "/"
		Filename: hashStr,                  // The file's final name is the full hash
	}
}

// PathKey method to get the first directory from the full path
func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/") // Split the path into parts
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
		PathName: key, // No hashing, just use the key as the path
		Filename: key, // No transformation, use key as filename
	}
}

// NewStore initializes a new Store with given options
func NewStore(opts StoreOpts) *Store {
	// Default function if none is provided
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	// Default root directory if none is provided
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// checks if a file exists in the store
func (s *Store) Has(id string, key string) bool {
	// Convert key to PathKey
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	// Check if the file exists
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

	// Log deletion after function execution
	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()

	// Construct path for the first directory to delete
	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

// writes data from an io.Reader into a file
func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

// writes encrypted data to a file
func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	// Open the file for writing
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	// Decrypt the data using copyDecrypt and write it to the file.
	n, err := copyDecrypt(encKey, r, f)

	// return the number of bytes written
	return int64(n), err
}

// openFileForWriting ensures the necessary directories exist and opens the file
func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	// Generate the path for the file using PathTransformFunc
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)

	// Create directories if they don't exist
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	// Create or overwrite the file
	return os.Create(fullPathWithRoot)
}

// writes data from an io.Reader to the file
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	// Copy data from reader to file
	return io.Copy(f, r)
}

// Reads data from a file in the store and returns its size and reader
func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

// readStream opens a file and returns its reader
func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	// Open the file for reading
	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	// Get file info and size
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	// Return file size and reader
	return fileInfo.Size(), file, nil
}
