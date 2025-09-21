package deduplication

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/testutil"
)

func TestDeduplicationManager(t *testing.T) {
	// Create a temporary vault directory
	vaultPath := testutil.TempDir(t, "dedup-test-vault")

	// Create vault structure
	err := os.MkdirAll(filepath.Join(vaultPath, ".sietch", "chunks"), 0o755)
	if err != nil {
		t.Fatalf("Failed to create vault structure: %v", err)
	}

	// Create deduplication config
	dedupConfig := config.DeduplicationConfig{
		Enabled:      true,
		Strategy:     "content",
		MinChunkSize: "0", // 0MB minimum = 0 bytes minimum for testing
		MaxChunkSize: "64",
		GCThreshold:  100,
		IndexEnabled: true,
	}

	// Create manager
	manager, err := NewManager(vaultPath, dedupConfig)
	if err != nil {
		t.Fatalf("Failed to create deduplication manager: %v", err)
	}

	// Test data
	testData := []byte("This is test chunk data for deduplication testing")
	chunkHash := "abc123def456"
	storageHash := "stored_abc123def456"

	// Create chunk reference
	chunkRef := config.ChunkRef{
		Hash:         chunkHash,
		Size:         int64(len(testData)),
		Index:        0,
		Compressed:   false,
		Deduplicated: false,
	}

	t.Run("ProcessNewChunk", func(t *testing.T) {
		// Process new chunk
		updatedRef, deduplicated, err := manager.ProcessChunk(chunkRef, testData, storageHash)
		if err != nil {
			t.Fatalf("Failed to process new chunk: %v", err)
		}

		if deduplicated {
			t.Error("New chunk should not be marked as deduplicated")
		}

		if updatedRef.Deduplicated {
			t.Error("New chunk reference should not be marked as deduplicated")
		}

		// Debug: Check if chunk exists in index
		if !manager.index.HasChunk(chunkHash) {
			t.Error("Chunk should exist in index after processing")
		}

		// Verify chunk exists in storage
		if !manager.ChunkExists(chunkHash) {
			t.Error("Chunk should exist after processing")
		}
	})

	t.Run("ProcessDuplicateChunk", func(t *testing.T) {
		// Process the same chunk again
		updatedRef, deduplicated, err := manager.ProcessChunk(chunkRef, testData, storageHash)
		if err != nil {
			t.Fatalf("Failed to process duplicate chunk: %v", err)
		}

		if !deduplicated {
			t.Error("Duplicate chunk should be marked as deduplicated")
		}

		if !updatedRef.Deduplicated {
			t.Error("Duplicate chunk reference should be marked as deduplicated")
		}

		// Verify chunk still exists
		if !manager.ChunkExists(chunkHash) {
			t.Error("Chunk should still exist after deduplication")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := manager.GetStats()

		if stats.TotalChunks != 1 {
			t.Errorf("Expected 1 total chunk, got %d", stats.TotalChunks)
		}

		if stats.TotalSize != int64(len(testData)) {
			t.Errorf("Expected total size %d, got %d", len(testData), stats.TotalSize)
		}

		if stats.SavedSpace != int64(len(testData)) {
			t.Errorf("Expected saved space %d, got %d", len(testData), stats.SavedSpace)
		}
	})

	t.Run("SaveAndLoad", func(t *testing.T) {
		// Save the index
		err := manager.Save()
		if err != nil {
			t.Fatalf("Failed to save index: %v", err)
		}

		// Create a new manager to test loading
		newManager, err := NewManager(vaultPath, dedupConfig)
		if err != nil {
			t.Fatalf("Failed to create new manager: %v", err)
		}

		// Verify the chunk exists in the new manager
		if !newManager.ChunkExists(chunkHash) {
			t.Error("Chunk should exist in loaded index")
		}

		// Verify stats are preserved
		stats := newManager.GetStats()
		if stats.TotalChunks != 1 {
			t.Errorf("Expected 1 total chunk after reload, got %d", stats.TotalChunks)
		}
	})
}

func TestDeduplicationIndex(t *testing.T) {
	// Create a temporary vault directory
	vaultPath := testutil.TempDir(t, "dedup-index-test")

	// Create index
	index, err := NewDeduplicationIndex(vaultPath)
	if err != nil {
		t.Fatalf("Failed to create deduplication index: %v", err)
	}

	// Test data
	chunkRef := config.ChunkRef{
		Hash:       "test_hash_123",
		Size:       1024,
		Index:      0,
		Compressed: false,
	}
	storageHash := "storage_test_hash_123"

	t.Run("AddNewChunk", func(t *testing.T) {
		entry, deduplicated := index.AddChunk(chunkRef, storageHash)

		if deduplicated {
			t.Error("New chunk should not be marked as deduplicated")
		}

		if entry.RefCount != 1 {
			t.Errorf("Expected ref count 1, got %d", entry.RefCount)
		}

		if entry.Hash != chunkRef.Hash {
			t.Errorf("Expected hash %s, got %s", chunkRef.Hash, entry.Hash)
		}
	})

	t.Run("AddDuplicateChunk", func(t *testing.T) {
		entry, deduplicated := index.AddChunk(chunkRef, storageHash)

		if !deduplicated {
			t.Error("Duplicate chunk should be marked as deduplicated")
		}

		if entry.RefCount != 2 {
			t.Errorf("Expected ref count 2, got %d", entry.RefCount)
		}
	})

	t.Run("HasChunk", func(t *testing.T) {
		if !index.HasChunk(chunkRef.Hash) {
			t.Error("Index should contain the chunk")
		}

		if index.HasChunk("nonexistent_hash") {
			t.Error("Index should not contain nonexistent chunk")
		}
	})

	t.Run("GetChunk", func(t *testing.T) {
		entry, exists := index.GetChunk(chunkRef.Hash)

		if !exists {
			t.Error("Chunk should exist in index")
		}

		if entry.Hash != chunkRef.Hash {
			t.Errorf("Expected hash %s, got %s", chunkRef.Hash, entry.Hash)
		}

		if entry.RefCount != 2 {
			t.Errorf("Expected ref count 2, got %d", entry.RefCount)
		}
	})

	t.Run("RemoveChunk", func(t *testing.T) {
		// Remove chunk once (should decrement ref count)
		err := index.RemoveChunk(chunkRef.Hash)
		if err != nil {
			t.Fatalf("Failed to remove chunk: %v", err)
		}

		entry, exists := index.GetChunk(chunkRef.Hash)
		if !exists {
			t.Error("Chunk should still exist after single removal")
		}

		if entry.RefCount != 1 {
			t.Errorf("Expected ref count 1 after removal, got %d", entry.RefCount)
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := index.GetStats()

		if stats.TotalChunks != 1 {
			t.Errorf("Expected 1 total chunk, got %d", stats.TotalChunks)
		}

		if stats.TotalSize != chunkRef.Size {
			t.Errorf("Expected total size %d, got %d", chunkRef.Size, stats.TotalSize)
		}
	})
}
