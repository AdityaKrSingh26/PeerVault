package main

import (
	"bytes"
	"fmt"
	"testing"
)

// Steps:
// Define the Payload
// Generate an Encryption Key
// Encrypt the Data
// Print Debug Information
// Decrypt the Encrypted Data
// Check the Number of Bytes Written
// Validate Decryption

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo not bar"
	// src creates an io.Reader from payload (acts as a file or input stream).
	src := bytes.NewReader([]byte(payload))
	// dst is an io.Writer (bytes.Buffer) to store the encrypted output
	dst := new(bytes.Buffer)

	// generates a random 32-byte AES key.
	key := newEncryptionKey()

	// Calls copyEncrypt to encrypt payload, storing the result in dst
	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))

	// Decrypt the Encrypted Data
	out := new(bytes.Buffer)
	nw, err := copyDecrypt(key, dst, out)
	if err != nil {
		t.Error(err)
	}

	// Expected size = IV (16 bytes) + original payload size.
	if nw != 16+len(payload) {
		t.Fail()
	}

	if out.String() != payload {
		t.Errorf("decryption failed!!!")
	}
}
