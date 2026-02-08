package storage

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/AdityaKrSingh26/PeerVault/internal/crypto"
)

func TestPathTransformFunc(t *testing.T) {
	key := "momsbestpicture"
	pathKey := CASPathTransformFunc(key)
	// SHA-256 hash of "momsbestpicture" is b159a9f0a78305c07dbce386598952bfa30b6aabb46a98b072c9195348abf9ea
	expectedFilename := "b159a9f0a78305c07dbce386598952bfa30b6aabb46a98b072c9195348abf9ea"
	expectedPathName := "b159a/9f0a7/8305c/07dbc/e3865/98952/bfa30/b6aab/b46a9/8b072/c9195/348ab"
	if pathKey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathKey.PathName, expectedPathName)
	}

	if pathKey.Filename != expectedFilename {
		t.Errorf("have %s want %s", pathKey.Filename, expectedFilename)
	}
}

// Tests the core functionality of the Store struct, including:
// Writing files.
// Checking if files exist.
// Reading files.
// Deleting files.

func TestStore(t *testing.T) {
	s := newStore()
	id := crypto.GenerateID()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")

		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("want %s have %s", data, b)
		}

		if err := s.Delete(id, key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); ok {
			t.Errorf("expected to NOT have key %s", key)
		}
	}
}

// initializes a new Store with the CAS path transformation function
func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

// clears the store after a test completes by deleting all files and directories.
func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
