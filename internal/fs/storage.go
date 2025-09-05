package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// StoreChunk writes a chunk to the chunk storage with the given hash as filename
func StoreChunk(basePath string, chunkHash string, data []byte) error {
	chunkPath := filepath.Join(GetChunkDirectory(basePath), chunkHash)

	// Write the chunk data to file
	if err := os.WriteFile(chunkPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write chunk %s: %w", chunkHash, err)
	}

	return nil
}

// ChunkExists checks if a chunk with the given hash exists
func ChunkExists(basePath string, chunkHash string) bool {
	chunkPath := filepath.Join(GetChunkDirectory(basePath), chunkHash)
	_, err := os.Stat(chunkPath)
	return err == nil
}

// GetChunk retrieves a chunk by its hash
func GetChunk(basePath string, chunkHash string) ([]byte, error) {
	chunkPath := filepath.Join(GetChunkDirectory(basePath), chunkHash)

	data, err := os.ReadFile(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk %s: %w", chunkHash, err)
	}

	return data, nil
}
