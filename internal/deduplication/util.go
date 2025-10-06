package deduplication

import (
	"sort"

	"github.com/substantialcattle5/sietch/internal/config"
)

// ComputeDedupStatsForFile calculates dedup stats by consulting chunkRefs map.
// Uses EncryptedSize if present, otherwise Size, otherwise falls back to default chunk size.
func ComputeDedupStatsForFile(file config.FileManifest, chunkRefs map[string][]string) (sharedChunks int, savedBytes int64, sharedWith []string) {
	// Default chunk size assumption (matches docs): 4 MiB
	const defaultChunkSize int64 = 4 * 1024 * 1024

	sharedWithSet := make(map[string]struct{})
	filePath := file.Destination + file.FilePath

	for _, c := range file.Chunks {
		chunkID := c.Hash
		if chunkID == "" {
			chunkID = c.EncryptedHash
		}
		if chunkID == "" {
			continue
		}

		refs, ok := chunkRefs[chunkID]
		if !ok {
			continue
		}
		if len(refs) > 1 {
			sharedChunks++

			// Prefer encrypted size if available (actual stored size), fallback to plaintext size
			var chunkSize int64
			if c.EncryptedSize > 0 {
				chunkSize = c.EncryptedSize
			} else if c.Size > 0 {
				chunkSize = c.Size
			} else {
				chunkSize = defaultChunkSize
			}
			savedBytes += chunkSize

			for _, other := range refs {
				if other == filePath {
					continue
				}
				sharedWithSet[other] = struct{}{}
			}
		}
	}

	sharedWith = make([]string, 0, len(sharedWithSet))
	for s := range sharedWithSet {
		sharedWith = append(sharedWith, s)
	}
	// sort for deterministic output
	sort.Strings(sharedWith)
	return
}
