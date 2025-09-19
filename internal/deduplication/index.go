package deduplication

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/fs"
)

// NewDeduplicationIndex creates a new deduplication index
func NewDeduplicationIndex(vaultRoot string) (*DeduplicationIndex, error) {
	indexPath := filepath.Join(vaultRoot, ".sietch", "dedup_index.json")

	idx := &DeduplicationIndex{
		vaultRoot: vaultRoot,
		indexPath: indexPath,
		entries:   make(map[string]*ChunkIndexEntry),
		dirty:     false,
	}

	// Load existing index if it exists
	if err := idx.Load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it when we save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load deduplication index: %w", err)
		}
	}

	return idx, nil
}

// Load loads the deduplication index from disk
func (idx *DeduplicationIndex) Load() error {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	data, err := os.ReadFile(idx.indexPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &idx.entries)
}

// Save saves the deduplication index to disk
func (idx *DeduplicationIndex) Save() error {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	if !idx.dirty {
		return nil // No changes to save
	}

	// Ensure the directory exists
	dir := filepath.Dir(idx.indexPath)
	if err := os.MkdirAll(dir, constants.StandardDirPerms); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	data, err := json.MarshalIndent(idx.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(idx.indexPath, data, constants.StandardFilePerms); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	idx.dirty = false
	return nil
}

// HasChunk checks if a chunk exists in the index
func (idx *DeduplicationIndex) HasChunk(hash string) bool {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	_, exists := idx.entries[hash]
	return exists
}

// GetChunk retrieves chunk metadata from the index
func (idx *DeduplicationIndex) GetChunk(hash string) (*ChunkIndexEntry, bool) {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	entry, exists := idx.entries[hash]
	if !exists {
		return nil, false
	}

	// Create a copy to avoid race conditions
	entryCopy := *entry
	return &entryCopy, true
}

// AddChunk adds a new chunk to the index or increments reference count if it exists
func (idx *DeduplicationIndex) AddChunk(chunkRef config.ChunkRef, storageHash string) (*ChunkIndexEntry, bool) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	now := time.Now()

	// Check if chunk already exists
	if entry, exists := idx.entries[chunkRef.Hash]; exists {
		// Increment reference count
		entry.RefCount++
		entry.LastReferenced = now
		idx.dirty = true

		// Create a copy to return
		entryCopy := *entry
		return &entryCopy, true // true indicates deduplication occurred
	}

	// Create new entry
	entry := &ChunkIndexEntry{
		Hash:           chunkRef.Hash,
		Size:           chunkRef.Size,
		RefCount:       1,
		StorageHash:    storageHash,
		FirstSeen:      now,
		LastReferenced: now,
		Compressed:     chunkRef.Compressed,
		Encrypted:      chunkRef.EncryptedHash != "",
	}

	idx.entries[chunkRef.Hash] = entry
	idx.dirty = true

	// Create a copy to return
	entryCopy := *entry
	return &entryCopy, false // false indicates new chunk
}

// RemoveChunk decrements the reference count of a chunk and removes it if ref count reaches 0
func (idx *DeduplicationIndex) RemoveChunk(hash string) error {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	entry, exists := idx.entries[hash]
	if !exists {
		return fmt.Errorf("chunk not found in index: %s", hash)
	}

	entry.RefCount--
	if entry.RefCount <= 0 {
		delete(idx.entries, hash)
		idx.dirty = true

		// Also remove the actual chunk file
		return idx.removeChunkFile(entry.StorageHash)
	}

	idx.dirty = true
	return nil
}

// removeChunkFile removes the physical chunk file from storage
func (idx *DeduplicationIndex) removeChunkFile(storageHash string) error {
	chunkPath := filepath.Join(fs.GetChunkDirectory(idx.vaultRoot), storageHash)
	if err := os.Remove(chunkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove chunk file %s: %w", storageHash, err)
	}
	return nil
}

// GetStats returns statistics about the deduplication index
func (idx *DeduplicationIndex) GetStats() DeduplicationStats {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	stats := DeduplicationStats{
		TotalChunks:        len(idx.entries),
		TotalSize:          0,
		UnreferencedChunks: 0,
		SavedSpace:         0,
	}

	for _, entry := range idx.entries {
		stats.TotalSize += entry.Size
		if entry.RefCount == 0 {
			stats.UnreferencedChunks++
		}
		if entry.RefCount > 1 {
			stats.SavedSpace += entry.Size * int64(entry.RefCount-1)
		}
	}

	return stats
}

// GarbageCollect removes unreferenced chunks
func (idx *DeduplicationIndex) GarbageCollect() (int, error) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	var toRemove []string
	for hash, entry := range idx.entries {
		if entry.RefCount <= 0 {
			toRemove = append(toRemove, hash)
		}
	}

	for _, hash := range toRemove {
		entry := idx.entries[hash]
		if err := idx.removeChunkFile(entry.StorageHash); err != nil {
			fmt.Printf("Warning: failed to remove chunk file for %s: %v\n", hash, err)
		}
		delete(idx.entries, hash)
	}

	if len(toRemove) > 0 {
		idx.dirty = true
	}

	return len(toRemove), nil
}
