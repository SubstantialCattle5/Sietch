package deduplication

import (
	"fmt"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/atomic"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/util"
)

// Manager handles deduplication operations for a vault
type Manager struct {
	vaultRoot   string
	config      config.DeduplicationConfig
	index       *DeduplicationIndex
	progressMgr ProgressManager
}

// ProgressManager is an interface for progress reporting
type ProgressManager interface {
	PrintVerbose(format string, args ...interface{})
}

// NewManager creates a new deduplication manager
func NewManager(vaultRoot string, dedupConfig config.DeduplicationConfig) (*Manager, error) {
	index, err := NewDeduplicationIndex(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create deduplication index: %w", err)
	}

	return &Manager{
		vaultRoot:   vaultRoot,
		config:      dedupConfig,
		index:       index,
		progressMgr: nil, // Will be set later if needed
	}, nil
}

// SetProgressManager sets the progress manager for verbose output
func (m *Manager) SetProgressManager(pm ProgressManager) {
	m.progressMgr = pm
}

// ProcessChunk processes a chunk for deduplication
// Returns: (chunkRef, deduplicated, error)
func (m *Manager) ProcessChunk(chunkRef config.ChunkRef, chunkData []byte, storageHash string) (config.ChunkRef, bool, error) {
	if !m.config.Enabled {
		// Deduplication disabled, store chunk normally
		if err := m.storeChunk(storageHash, chunkData); err != nil {
			return chunkRef, false, err
		}
		return chunkRef, false, nil
	}

	// Check if we should deduplicate this chunk based on size constraints
	if !m.shouldDeduplicateChunk(chunkRef.Size) {
		// Store chunk normally without deduplication
		if err := m.storeChunk(storageHash, chunkData); err != nil {
			return chunkRef, false, err
		}
		return chunkRef, false, nil
	}

	// Check if chunk already exists in index
	entry, deduplicated := m.index.AddChunk(chunkRef, storageHash)

	if deduplicated {
		// Chunk already exists, no need to store it again
		chunkRef.Deduplicated = true
		if m.progressMgr != nil {
			m.progressMgr.PrintVerbose("  └─ Deduplicated chunk %s (ref count: %d)\n",
				chunkRef.Hash[:12], entry.RefCount)
		}
	} else {
		// New chunk, store it
		if err := m.storeChunk(storageHash, chunkData); err != nil {
			// Remove from index if storage failed
			if m.index.RemoveChunk(chunkRef.Hash) != nil {
				fmt.Printf("Warning: failed to remove chunk %s from index: %v\n", chunkRef.Hash, err)
			}
			return chunkRef, false, err
		}
		chunkRef.Deduplicated = false
	}

	return chunkRef, deduplicated, nil
}

// shouldDeduplicateChunk checks if a chunk should be deduplicated based on configuration
func (m *Manager) shouldDeduplicateChunk(chunkSize int64) bool {
	if !m.config.Enabled {
		return false
	}

	minSize, err := util.ParseChunkSize(m.config.MinChunkSize)
	if err != nil {
		minSize = 1024 // Default to 1KB
	}

	maxSize, err := util.ParseChunkSize(m.config.MaxChunkSize)
	if err != nil {
		maxSize = 64 * 1024 * 1024 // Default to 64MB
	}

	return chunkSize >= minSize && chunkSize <= maxSize
}

// storeChunk stores a chunk to the filesystem
func (m *Manager) storeChunk(storageHash string, chunkData []byte) error {
	return fs.StoreChunk(m.vaultRoot, storageHash, chunkData)
}

// storeChunkTransactional stages a chunk into the active transaction instead of writing directly.
func (m *Manager) storeChunkTransactional(txn *atomic.Transaction, storageHash string, chunkData []byte) error {
	rel := filepath.ToSlash(filepath.Join(".sietch", "chunks", storageHash))
	w, err := txn.StageCreate(rel)
	if err != nil {
		return fmt.Errorf("stage chunk %s: %w", storageHash, err)
	}
	if _, err := w.Write(chunkData); err != nil {
		_ = w.Close()
		return fmt.Errorf("write staged chunk %s: %w", storageHash, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close staged chunk %s: %w", storageHash, err)
	}
	return nil
}

// ProcessChunkTransactional mirrors ProcessChunk but stores new chunk content via the transaction staging area.
func (m *Manager) ProcessChunkTransactional(txn *atomic.Transaction, chunkRef config.ChunkRef, chunkData []byte, storageHash string) (config.ChunkRef, bool, error) {
	if !m.config.Enabled {
		if err := m.storeChunkTransactional(txn, storageHash, chunkData); err != nil {
			return chunkRef, false, err
		}
		return chunkRef, false, nil
	}
	if !m.shouldDeduplicateChunk(chunkRef.Size) {
		if err := m.storeChunkTransactional(txn, storageHash, chunkData); err != nil {
			return chunkRef, false, err
		}
		return chunkRef, false, nil
	}
	entry, deduplicated := m.index.AddChunk(chunkRef, storageHash)
	if deduplicated {
		chunkRef.Deduplicated = true
		if m.progressMgr != nil {
			m.progressMgr.PrintVerbose("  └─ Deduplicated chunk %s (ref count: %d)\n", chunkRef.Hash[:12], entry.RefCount)
		}
		return chunkRef, true, nil
	}
	if err := m.storeChunkTransactional(txn, storageHash, chunkData); err != nil {
		if m.index.RemoveChunk(chunkRef.Hash) != nil {
			fmt.Printf("Warning: failed to remove chunk %s from index after transactional store failure\n", chunkRef.Hash)
		}
		return chunkRef, false, err
	}
	chunkRef.Deduplicated = false
	return chunkRef, false, nil
}

// GetStats returns deduplication statistics
func (m *Manager) GetStats() DeduplicationStats {
	return m.index.GetStats()
}

// GarbageCollect removes unreferenced chunks
func (m *Manager) GarbageCollect() (int, error) {
	return m.index.GarbageCollect()
}

// Save saves the deduplication index
func (m *Manager) Save() error {
	return m.index.Save()
}

// RemoveFileChunks removes all chunks associated with a file
func (m *Manager) RemoveFileChunks(chunks []config.ChunkRef) error {
	for _, chunk := range chunks {
		if err := m.index.RemoveChunk(chunk.Hash); err != nil {
			return fmt.Errorf("failed to remove chunk %s: %w", chunk.Hash, err)
		}
	}
	return nil
}

// OptimizeStorage performs optimization operations
func (m *Manager) OptimizeStorage() (*OptimizationResult, error) {
	stats := m.GetStats()

	// Perform garbage collection
	removedChunks, err := m.GarbageCollect()
	if err != nil {
		return nil, fmt.Errorf("garbage collection failed: %w", err)
	}

	// Save index after optimization
	if err := m.Save(); err != nil {
		return nil, fmt.Errorf("failed to save index after optimization: %w", err)
	}

	return &OptimizationResult{
		RemovedChunks:      removedChunks,
		TotalChunks:        stats.TotalChunks,
		SavedSpace:         stats.SavedSpace,
		UnreferencedChunks: stats.UnreferencedChunks,
	}, nil
}

// OptimizationResult contains the results of storage optimization
type OptimizationResult struct {
	RemovedChunks      int   `json:"removed_chunks"`
	TotalChunks        int   `json:"total_chunks"`
	SavedSpace         int64 `json:"saved_space"`
	UnreferencedChunks int   `json:"unreferenced_chunks"`
}

// ChunkExists checks if a chunk exists (for compatibility with existing code)
func (m *Manager) ChunkExists(hash string) bool {
	if !m.config.Enabled {
		return fs.ChunkExists(m.vaultRoot, hash)
	}
	return m.index.HasChunk(hash)
}

// GetChunk retrieves a chunk (for compatibility with existing code)
func (m *Manager) GetChunk(hash string) ([]byte, error) {
	if !m.config.Enabled {
		return fs.GetChunk(m.vaultRoot, hash)
	}

	// Get chunk metadata from index
	entry, exists := m.index.GetChunk(hash)
	if !exists {
		return nil, fmt.Errorf("chunk not found: %s", hash)
	}

	// Retrieve chunk using storage hash
	return fs.GetChunk(m.vaultRoot, entry.StorageHash)
}
