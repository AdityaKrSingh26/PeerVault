package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// Generates a random 32-byte ID and encodes it as a hexadecimal string
// generating unique identifiers
func generateID() string {
	buf := make([]byte, 32)
	// Fills the slice with cryptographically secure random bytes
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

// Hashes a given key using the MD5 algorithm
func hashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// Generates a random 32-byte encryption key.
func newEncryptionKey() []byte {
	// Create a 32-byte buffer
	keyBuf := make([]byte, 32)
	// Fill buffer with cryptographically secure random bytes
	// ensures the buffer is completely filled with 32 random bytes.
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

// Copies data from a source (src) to a destination (dst) while applying a stream cipher
// Used by copyEncrypt and copyDecrypt to handle the actual encryption/decryption process
func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	// Creates a buffer (buf) of size 32 KB to read data in chunks
	buf := make([]byte, 32*1024)
	nw := blockSize

	for {
		// Read up to 32KB from source(src) into buffer
		n, err := src.Read(buf)
		if n > 0 {
			// Applies the stream cipher to the buffer using stream cipher
			stream.XORKeyStream(buf, buf[:n])
			// Write encrypted/decrypted data to dst
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			// Track bytes written
			nw += nn
		}
		if err == io.EOF {
			// If source ends, stop loop
			break
		}
		if err != nil {
			return 0, err
		}
	}
	// return total bytes written
	return nw, nil
}

// Decrypts data from src and writes the decrypted data to dst
// Used to decrypt data that was encrypted using copyEncrypt
func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	// Creates an AES cipher block using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Reads the Initialization Vector(Prevents pattern recognition,Ensures uniqueness)
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	// CTR (Counter Mode) is an encryption mode that turns a block cipher into a stream cipher
	stream := cipher.NewCTR(block, iv) 

	// copyStream to decrypt the data.
	return copyStream(stream, block.BlockSize(), src, dst)
}

// Encrypts data from src and writes the encrypted data to dst
// Used to encrypt data for secure storage or transmission
func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	// Creates an AES cipher block using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// Generates a random Initialization Vector (IV)
	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// prepend the IV to the file.
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	// Calls copyStream to encrypt the data
	return copyStream(stream, block.BlockSize(), src, dst)
}
