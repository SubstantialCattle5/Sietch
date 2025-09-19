package deduplication

import (
	"sync"
	"time"
)

// DeduplicationStats contains statistics about deduplication
type DeduplicationStats struct {
	TotalChunks        int   `json:"total_chunks"`
	TotalSize          int64 `json:"total_size"`
	UnreferencedChunks int   `json:"unreferenced_chunks"`
	SavedSpace         int64 `json:"saved_space"`
}

// ChunkIndexEntry represents metadata about a chunk in the deduplication index
type ChunkIndexEntry struct {
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	RefCount       int       `json:"ref_count"`
	StorageHash    string    `json:"storage_hash"` // Hash used for storage (encrypted hash if applicable)
	FirstSeen      time.Time `json:"first_seen"`
	LastReferenced time.Time `json:"last_referenced"`
	Compressed     bool      `json:"compressed"`
	Encrypted      bool      `json:"encrypted"`
}

// DeduplicationIndex manages the chunk deduplication index
type DeduplicationIndex struct {
	vaultRoot string
	indexPath string
	entries   map[string]*ChunkIndexEntry
	mutex     sync.RWMutex
	dirty     bool // Track if index needs to be saved
}
