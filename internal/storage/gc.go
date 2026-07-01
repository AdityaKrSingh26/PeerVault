package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
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
	logger           *slog.Logger
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(store *Store, nodeID string, logger *slog.Logger) *GarbageCollector {
	if logger == nil {
		logger = slog.Default()
	}
	if err := ValidateNodeID(nodeID); err != nil {
		logger.Error("invalid node ID for garbage collector", "node", nodeID, "err", err)
		os.Exit(1)
	}
	return &GarbageCollector{
		store:            store,
		nodeID:           nodeID,
		cleanupInterval:  1 * time.Hour, // Run cleanup every hour
		integrityEnabled: true,
		stopChan:         make(chan struct{}),
		logger:           logger,
	}
}

// Start begins the periodic garbage collection routine
func (gc *GarbageCollector) Start(ctx context.Context) {
	gc.logger.Info("Starting garbage collector", "node", gc.nodeID)
	go gc.run(ctx)
}

// Stop stops the garbage collection routine
func (gc *GarbageCollector) Stop() {
	close(gc.stopChan)
}

// run is the main garbage collection loop
func (gc *GarbageCollector) run(ctx context.Context) {
	ticker := time.NewTicker(gc.cleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup after 5 minutes
	initialDelay := time.NewTimer(5 * time.Minute)
	defer initialDelay.Stop()

	for {
		select {
		case <-initialDelay.C:
			gc.performCleanup()
		case <-ticker.C:
			gc.performCleanup()
		case <-ctx.Done():
			gc.logger.Info("Garbage collector stopped due to context cancellation", "node", gc.nodeID)
			return
		case <-gc.stopChan:
			gc.logger.Info("Garbage collector stopped", "node", gc.nodeID)
			return
		}
	}
}

// performCleanup runs integrity checks and cleanup operations
func (gc *GarbageCollector) performCleanup() {
	gc.logger.Info("Running garbage collection", "node", gc.nodeID)
	start := time.Now()

	stats := CleanupStats{
		CorruptedFiles: 0,
		OrphanedFiles:  0,
		RemovedFiles:   0,
	}

	if gc.integrityEnabled {
		// Verify file integrity
		if err := gc.verifyIntegrity(&stats); err != nil {
			gc.logger.Error("Error during integrity verification", "node", gc.nodeID, "err", err)
		}
	}

	// Clean up orphaned files
	if err := gc.cleanOrphanedFiles(&stats); err != nil {
		gc.logger.Error("Error during orphan cleanup", "node", gc.nodeID, "err", err)
	}

	elapsed := time.Since(start)
	gc.logger.Info("Garbage collection completed",
		"node", gc.nodeID,
		"duration", elapsed,
		"corrupted", stats.CorruptedFiles,
		"orphaned", stats.OrphanedFiles,
		"removed", stats.RemovedFiles,
	)
}

// CleanupStats tracks garbage collection statistics
type CleanupStats struct {
	CorruptedFiles int
	OrphanedFiles  int
	RemovedFiles   int
}

// verifyIntegrity checks if stored files have valid hashes
func (gc *GarbageCollector) verifyIntegrity(stats *CleanupStats) error {
	gc.logger.Info("Verifying file integrity", "node", gc.nodeID)

	nodeDir, err := gc.store.resolvePath(gc.nodeID, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil // No files to check
	}

	err = filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			gc.logger.Warn("walk error", "node", gc.nodeID, "path", path, "err", err)
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
			gc.logger.Warn("Failed to calculate hash", "node", gc.nodeID, "path", path, "err", err)
			return nil
		}

		// Compare hashes
		if actualHash != expectedHash {
			gc.logger.Error("INTEGRITY VIOLATION: File has incorrect hash",
				"node", gc.nodeID,
				"path", path,
				"expected", expectedHash,
				"actual", actualHash,
			)
			stats.CorruptedFiles++

			// Remove corrupted file
			if err := os.RemoveAll(filepath.Dir(path)); err != nil {
				gc.logger.Error("Failed to remove corrupted file", "node", gc.nodeID, "path", path, "err", err)
			} else {
				gc.logger.Info("Removed corrupted file", "node", gc.nodeID, "path", path)
				stats.RemovedFiles++
			}
		}

		return nil
	})

	return err
}

// cleanOrphanedFiles removes empty directories and temporary files
func (gc *GarbageCollector) cleanOrphanedFiles(stats *CleanupStats) error {
	gc.logger.Info("Cleaning orphaned files", "node", gc.nodeID)

	nodeDir, err := gc.store.resolvePath(gc.nodeID, "")
	if err != nil {
		return err
	}
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil
	}

	// Find and remove empty directories
	err = filepath.Walk(nodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			gc.logger.Warn("walk error", "node", gc.nodeID, "path", path, "err", err)
			return nil
		}

		if info.IsDir() && path != nodeDir {
			// Check if directory is empty
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			if len(entries) == 0 {
				gc.logger.Info("Removing empty directory", "node", gc.nodeID, "path", path)
				if err := os.Remove(path); err != nil {
					gc.logger.Error("Failed to remove empty directory", "node", gc.nodeID, "path", path, "err", err)
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
	fullPath, err := gc.store.resolvePath(gc.nodeID, pathKey.FullPath())
	if err != nil {
		return false, err
	}

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
