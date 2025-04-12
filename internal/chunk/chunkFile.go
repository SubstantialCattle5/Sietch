package chunk

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
)

func ChunkFile(filePath string, chunkSize int64, vaultRoot string) ([]config.ChunkRef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found at %s", filePath)
		}
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Ensure chunks directory exists
	chunksDir := fs.GetChunkDirectory(vaultRoot)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Create a buffer for reading chunks
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	totalBytes := int64(0)
	chunkRefs := []config.ChunkRef{}

	// Read the file in chunks
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading file: %v", err)
		}

		if bytesRead == 0 {
			// End of file
			break
		}

		chunkCount++
		totalBytes += int64(bytesRead)

		// calculate chunk hash
		hasher := sha256.New()
		hasher.Write(buffer[:bytesRead])
		chunkHash := fmt.Sprintf("%x", hasher.Sum(nil))

		fmt.Printf("Chunk %d hash: %s\n", chunkCount, chunkHash)

		// Save the chunk to the vault
		chunkPath := filepath.Join(chunksDir, chunkHash)

		//todo implement encryption later
		if err := os.WriteFile(chunkPath, buffer[:bytesRead], 0644); err != nil {
			return nil, fmt.Errorf("failed to write chunk file: %v", err)
		}
		fmt.Printf("Chunk %d: %s bytes, hash: %s\n", chunkCount, humanReadableSize(int64(bytesRead)), chunkHash)

		// Add the chunk reference to our list
		chunkRefs = append(chunkRefs, config.ChunkRef{
			Hash:  chunkHash,
			Size:  int64(bytesRead),
			Index: chunkCount - 1, // 0-based index
		})

		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Total chunks processed: %d\n", chunkCount)
	fmt.Printf("Total bytes processed: %s\n", humanReadableSize(totalBytes))

	return chunkRefs, nil
}

func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
