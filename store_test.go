package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "momsbestpicture"
	pathKey := CASPathTransformFunc(key)
	expectedFilename := "6804429f74181a63c50c3d81d733a12f14a353ff"
	expectedPathName := "68044/29f74/181a6/3c50c/3d81d/733a1/2f14a/353ff"
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
	id := generateID()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")

		// Write data to the store
		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		// Check if the file exists
		if ok := s.Has(id, key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		// Read the file
		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		// Compare the read data with the original data
		b, _ := io.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("want %s have %s", data, b)
		}

		// Delete the file
		if err := s.Delete(id, key); err != nil {
			t.Error(err)
		}

		// Verify that the file no longer exists
		if ok := s.Has(id, key); ok {
			t.Errorf("expected to NOT have key %s", key)
		}
	}
}

// initializes a new Store with the CAS path transformation function
func newStore() *Store {
	opts := StoreOpts{
		// Use content-addressable storage function
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
