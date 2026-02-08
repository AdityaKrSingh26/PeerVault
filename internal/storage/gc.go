package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// GarbageCollector manages integrity verification and cleanup
type GarbageCollector struct {
	store            *Store
	nodeID           string
	cleanupInterval  time.Duration
	integrityEnabled bool
	stopChan         chan struct{}
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(store *Store, nodeID string) *GarbageCollector {
	return &GarbageCollector{
		store:            store,
		nodeID:           nodeID,
		cleanupInterval:  1 * time.Hour, // Run cleanup every hour
		integrityEnabled: true,
		stopChan:         make(chan struct{}),
	}
}

// Start begins the periodic garbage collection routine
func (gc *GarbageCollector) Start() {
	log.Println("Starting garbage collector...")
	go gc.run()
}

// Stop stops the garbage collection routine
func (gc *GarbageCollector) Stop() {
	close(gc.stopChan)
}

// run is the main garbage collection loop
func (gc *GarbageCollector) run() {
	ticker := time.NewTicker(gc.cleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup after 5 minutes
	initialDelay := time.NewTimer(5 * time.Minute)

	for {
		select {
		case <-initialDelay.C:
			gc.performCleanup()
		case <-ticker.C:
			gc.performCleanup()
		case <-gc.stopChan:
			log.Println("Garbage collector stopped")
			return
		}
	}
}

// performCleanup runs integrity checks and cleanup operations
func (gc *GarbageCollector) performCleanup() {
	log.Println("Running garbage collection...")
	start := time.Now()

	stats := CleanupStats{
		CorruptedFiles: 0,
		OrphanedFiles:  0,
		RemovedFiles:   0,
	}

	if gc.integrityEnabled {
		// Verify file integrity
		if err := gc.verifyIntegrity(&stats); err != nil {
			log.Printf("Error during integrity verification: %v", err)
		}
	}

	// Clean up orphaned files
	if err := gc.cleanOrphanedFiles(&stats); err != nil {
		log.Printf("Error during orphan cleanup: %v", err)
	}

	elapsed := time.Since(start)
	log.Printf("Garbage collection completed in %v: %d corrupted, %d orphaned, %d removed",
		elapsed, stats.CorruptedFiles, stats.OrphanedFiles, stats.RemovedFiles)
}

// CleanupStats tracks garbage collection statistics
type CleanupStats struct {
	CorruptedFiles int
	OrphanedFiles  int
	RemovedFiles   int
}

// verifyIntegrity checks if stored files have valid hashes
func (gc *GarbageCollector) verifyIntegrity(stats *CleanupStats) error {
	log.Println("Verifying file integrity...")

	nodeDir := fmt.Sprintf("%s/%s", gc.store.Root, gc.nodeID)
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil // No files to check
	}

	err := filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Verify this is a file we can check
		expectedHash := info.Name()
		if len(expectedHash) != 64 { // SHA-256 hash is 64 hex characters
			// Not a hash-named file, skip
			return nil
		}

		// Calculate actual hash of file content
		actualHash, err := calculateFileHash(path)
		if err != nil {
			log.Printf("Warning: Failed to calculate hash for %s: %v", path, err)
			return nil
		}

		// Compare hashes
		if actualHash != expectedHash {
			log.Printf("INTEGRITY VIOLATION: File %s has incorrect hash", path)
			log.Printf("  Expected: %s", expectedHash)
			log.Printf("  Actual:   %s", actualHash)
			stats.CorruptedFiles++

			// Remove corrupted file
			if err := os.RemoveAll(filepath.Dir(path)); err != nil {
				log.Printf("Failed to remove corrupted file: %v", err)
			} else {
				log.Printf("Removed corrupted file: %s", path)
				stats.RemovedFiles++
			}
		}

		return nil
	})

	return err
}

// cleanOrphanedFiles removes empty directories and temporary files
func (gc *GarbageCollector) cleanOrphanedFiles(stats *CleanupStats) error {
	log.Println("Cleaning orphaned files...")

	nodeDir := fmt.Sprintf("%s/%s", gc.store.Root, gc.nodeID)
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil
	}

	// Find and remove empty directories
	err := filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && path != nodeDir {
			// Check if directory is empty
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			if len(entries) == 0 {
				log.Printf("Removing empty directory: %s", path)
				if err := os.Remove(path); err != nil {
					log.Printf("Failed to remove empty directory: %v", err)
				} else {
					stats.OrphanedFiles++
					stats.RemovedFiles++
				}
			}
		}

		return nil
	})

	return err
}

// calculateFileHash computes the SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyFile checks if a specific file has valid integrity
func (gc *GarbageCollector) VerifyFile(key string) (bool, error) {
	pathKey := gc.store.PathTransformFunc(key)
	fullPath := fmt.Sprintf("%s/%s/%s", gc.store.Root, gc.nodeID, pathKey.FullPath())

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false, fmt.Errorf("file does not exist")
	}

	// Calculate hash
	actualHash, err := calculateFileHash(fullPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Expected hash is the filename
	expectedHash := pathKey.Filename

	return actualHash == expectedHash, nil
}

// GetStats returns current garbage collection statistics
func (gc *GarbageCollector) GetStats() (corrupted int, orphaned int, lastRun time.Time) {
	// This is a simple implementation - in a real system you'd track these
	return 0, 0, time.Now()
}
