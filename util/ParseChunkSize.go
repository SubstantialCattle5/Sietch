package util

import "fmt"

func ParseChunkSize(chunkSize string) (int64, error) {
	var size int64
	_, err := fmt.Sscanf(chunkSize, "%d", &size)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", chunkSize)
	}

	return size * 1024 * 1024, nil
}
