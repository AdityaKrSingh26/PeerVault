package crypto

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
	key, err := NewEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}

	_, err = CopyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))

	out := new(bytes.Buffer)
	nw, err := CopyDecrypt(key, dst, out)
	if err != nil {
		t.Error(err)
	}

	// copyDecrypt should return the number of decrypted bytes written (not including IV)
	if nw != len(payload) {
		t.Errorf("Expected %d decrypted bytes, got %d", len(payload), nw)
	}

	if out.String() != payload {
		t.Errorf("decryption failed!!!")
	}
}

func TestHMACVerificationFailure(t *testing.T) {
	key, _ := NewEncryptionKey()
	payload := "Some test data to verify HMAC verification works"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)

	_, err := CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt a byte in the ciphertext portion (which is after HMAC [32 bytes] and IV [16 bytes])
	encryptedData := dst.Bytes()
	encryptedData[len(encryptedData)-1] ^= 0xFF

	out := new(bytes.Buffer)
	_, err = CopyDecrypt(key, bytes.NewReader(encryptedData), out)
	if err == nil {
		t.Error("Expected error due to corrupted ciphertext, but got nil")
	}
}

func TestWrongKeyDecryptionFailure(t *testing.T) {
	key1, _ := NewEncryptionKey()
	key2, _ := NewEncryptionKey()
	payload := "Secret message"
	
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)

	_, err := CopyEncrypt(key1, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	_, err = CopyDecrypt(key2, dst, out)
	if err == nil {
		t.Error("Expected error using wrong decryption key, but got nil")
	}
}

func TestZeroLengthInput(t *testing.T) {
	key, _ := NewEncryptionKey()
	src := bytes.NewReader([]byte{})
	dst := new(bytes.Buffer)

	_, err := CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	_, err = CopyDecrypt(key, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if out.Len() != 0 {
		t.Errorf("Expected empty output, got %d bytes", out.Len())
	}
}

func TestLargeInput(t *testing.T) {
	key, _ := NewEncryptionKey()
	// 100KB input, larger than 32KB buffer size
	payload := make([]byte, 100*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)

	_, err := CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	_, err = CopyDecrypt(key, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out.Bytes(), payload) {
		t.Error("Large input roundtrip failed - decrypted data does not match original")
	}
}
