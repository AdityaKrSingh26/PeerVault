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
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)

	// generates a random 32-byte AES key.
	key := newEncryptionKey()

	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))

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
