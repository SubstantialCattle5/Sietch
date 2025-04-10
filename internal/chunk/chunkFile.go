package chunk

import (
	"fmt"
	"io"
	"os"
)

func ChunkFile(filePath string, chunkSize int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found at %s", filePath)
		}
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create a buffer for reading chunks
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	totalBytes := int64(0)

	// Read the file in chunks
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading file: %v", err)
		}

		if bytesRead == 0 {
			// End of file
			break
		}

		chunkCount++
		totalBytes += int64(bytesRead)

		// Process the chunk (for now, just print its size and chunk number)
		fmt.Printf("Chunk %d: %s bytes\n", chunkCount, humanReadableSize(int64(bytesRead)))

		// TODO: Here we would do further processing:
		// 1. Calculate chunk hash
		// 2. Encrypt chunk if needed
		// 3. Save chunk to storage
		// 4. Add chunk reference to file manifest

		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Total chunks processed: %d\n", chunkCount)
	fmt.Printf("Total bytes processed: %s\n", humanReadableSize(totalBytes))

	return nil
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
