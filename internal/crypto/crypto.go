package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

// GenerateID generates unique identifiers safely, returning an error on entropy failure.
func GenerateID() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashKey hashes a given key using the SHA-256 algorithm
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// NewEncryptionKey generates a random 32-byte encryption key safely, returning an error on entropy failure.
func NewEncryptionKey() ([]byte, error) {
	keyBuf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, keyBuf); err != nil {
		return nil, err
	}
	return keyBuf, nil
}

// Copies data from a (src) to a (dst) while applying a stream cipher
func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	buf := make([]byte, 32*1024)
	nw := blockSize

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	// return total bytes written
	return nw, nil
}

func hmacKey(key []byte) []byte {
	h := sha256.New()
	h.Write(key)
	h.Write([]byte("peervault-hmac-v1"))
	return h.Sum(nil)
}

// CopyDecrypt decrypts data from src and writes the decrypted data to dst
// Used to decrypt data that was encrypted using CopyEncrypt
func CopyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	// 1. Read expected HMAC (32 bytes)
	expectedMac := make([]byte, 32)
	if _, err := io.ReadFull(src, expectedMac); err != nil {
		return 0, err
	}

	// 2. Read IV (16 bytes)
	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(src, iv); err != nil {
		return 0, err
	}

	// 3. Read the rest as ciphertext
	ciphertext, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}

	// 4. Recompute HMAC over [IV + ciphertext]
	h := hmac.New(sha256.New, hmacKey(key))
	h.Write(iv)
	h.Write(ciphertext)
	computedMac := h.Sum(nil)

	// 5. Compare HMACs in constant time
	if !hmac.Equal(expectedMac, computedMac) {
		return 0, errors.New("HMAC verification failed: ciphertext is corrupted or wrong key used")
	}

	// 6. Decrypt the ciphertext and write to dst
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	n, err := dst.Write(ciphertext)
	if err != nil {
		return 0, err
	}

	return n, nil
}

// CopyEncrypt encrypts data for secure storage or transmission
func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Encrypt to a temp buffer first so we can compute the HMAC over [IV + ciphertext]
	var ciphertextBuf bytes.Buffer
	stream := cipher.NewCTR(block, iv)

	// XORKeyStream to ciphertextBuf
	_, err = copyStream(stream, 0, src, &ciphertextBuf)
	if err != nil {
		return 0, err
	}

	ciphertext := ciphertextBuf.Bytes()

	// Compute HMAC-SHA256 over [IV + ciphertext]
	h := hmac.New(sha256.New, hmacKey(key))
	h.Write(iv)
	h.Write(ciphertext)
	mac := h.Sum(nil) // 32 bytes

	// Write HMAC (32 bytes) || IV (16 bytes) || ciphertext to dst
	if _, err := dst.Write(mac); err != nil {
		return 0, err
	}
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}
	if _, err := dst.Write(ciphertext); err != nil {
		return 0, err
	}

	return len(mac) + len(iv) + len(ciphertext), nil
}
