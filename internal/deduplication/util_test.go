package deduplication

import (
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
)

func TestComputeDedupStatsForFile(t *testing.T) {
	t.Run("FileWithSharedChunks", func(t *testing.T) {
		// Create a file manifest with chunks
		file := config.FileManifest{
			Destination: "/path/to/",
			FilePath:    "test.txt",
			Chunks: []config.ChunkRef{
				{Hash: "chunk1", Size: 1024},
				{Hash: "chunk2", Size: 2048},
				{Hash: "chunk3", Size: 512},
			},
		}

		// Create chunk reference map simulating shared chunks
		chunkRefs := map[string][]string{
			"chunk1": {"/path/to/test.txt", "/path/to/other1.txt"},                        // Shared
			"chunk2": {"/path/to/test.txt", "/path/to/other2.txt", "/path/to/other3.txt"}, // Shared with 2 others
			"chunk3": {"/path/to/test.txt"},                                               // Not shared
		}

		sharedChunks, savedBytes, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		// Should have 2 shared chunks (chunk1 and chunk2)
		if sharedChunks != 2 {
			t.Errorf("Expected 2 shared chunks, got %d", sharedChunks)
		}

		// Saved bytes should be chunk1 + chunk2 sizes
		expectedSavedBytes := int64(1024 + 2048)
		if savedBytes != expectedSavedBytes {
			t.Errorf("Expected saved bytes %d, got %d", expectedSavedBytes, savedBytes)
		}

		// Should be shared with other1.txt, other2.txt, other3.txt
		expectedSharedWith := []string{"/path/to/other1.txt", "/path/to/other2.txt", "/path/to/other3.txt"}
		if len(sharedWith) != len(expectedSharedWith) {
			t.Errorf("Expected %d shared with files, got %d", len(expectedSharedWith), len(sharedWith))
		}

		// Check that sharedWith is sorted (as per implementation)
		for i, expected := range expectedSharedWith {
			if i < len(sharedWith) && sharedWith[i] != expected {
				t.Errorf("Expected shared with file %s at index %d, got %s", expected, i, sharedWith[i])
			}
		}
	})

	t.Run("FileWithNoSharedChunks", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/unique/",
			FilePath:    "unique.txt",
			Chunks: []config.ChunkRef{
				{Hash: "unique_chunk1", Size: 500},
				{Hash: "unique_chunk2", Size: 1000},
			},
		}

		// No shared chunks - each chunk only references this file
		chunkRefs := map[string][]string{
			"unique_chunk1": {"/unique/unique.txt"},
			"unique_chunk2": {"/unique/unique.txt"},
		}

		sharedChunks, savedBytes, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		if sharedChunks != 0 {
			t.Errorf("Expected 0 shared chunks, got %d", sharedChunks)
		}

		if savedBytes != 0 {
			t.Errorf("Expected 0 saved bytes, got %d", savedBytes)
		}

		if len(sharedWith) != 0 {
			t.Errorf("Expected no shared files, got %v", sharedWith)
		}
	})

	t.Run("FileWithEncryptedChunks", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/encrypted/",
			FilePath:    "secret.txt",
			Chunks: []config.ChunkRef{
				{
					Hash:          "", // No plain hash
					EncryptedHash: "encrypted_chunk1",
					Size:          1024,
					EncryptedSize: 1100, // Slightly larger due to encryption
				},
				{
					Hash:          "plain_chunk2",
					EncryptedHash: "encrypted_chunk2",
					Size:          2048,
					EncryptedSize: 2200,
				},
			},
		}

		chunkRefs := map[string][]string{
			"encrypted_chunk1": {"/encrypted/secret.txt", "/encrypted/other_secret.txt"},
			"plain_chunk2":     {"/encrypted/secret.txt", "/encrypted/another.txt"},
		}

		sharedChunks, savedBytes, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		// Both chunks are shared
		if sharedChunks != 2 {
			t.Errorf("Expected 2 shared chunks, got %d", sharedChunks)
		}

		// Should use encrypted sizes when available
		expectedSavedBytes := int64(1100 + 2200) // EncryptedSize is preferred
		if savedBytes != expectedSavedBytes {
			t.Errorf("Expected saved bytes %d, got %d", expectedSavedBytes, savedBytes)
		}

		if len(sharedWith) != 2 {
			t.Errorf("Expected 2 shared files, got %d", len(sharedWith))
		}
	})

	t.Run("FileWithMissingChunkRefs", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/missing/",
			FilePath:    "incomplete.txt",
			Chunks: []config.ChunkRef{
				{Hash: "existing_chunk", Size: 1024},
				{Hash: "missing_chunk", Size: 2048},
			},
		}

		// Only one chunk exists in the reference map
		chunkRefs := map[string][]string{
			"existing_chunk": {"/missing/incomplete.txt", "/missing/other.txt"},
			// "missing_chunk" is not in the map
		}

		sharedChunks, savedBytes, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		// Only existing_chunk is shared
		if sharedChunks != 1 {
			t.Errorf("Expected 1 shared chunk, got %d", sharedChunks)
		}

		expectedSavedBytes := int64(1024)
		if savedBytes != expectedSavedBytes {
			t.Errorf("Expected saved bytes %d, got %d", expectedSavedBytes, savedBytes)
		}

		if len(sharedWith) != 1 {
			t.Errorf("Expected 1 shared file, got %d", len(sharedWith))
		}
	})

	t.Run("FileWithEmptyHashes", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/empty/",
			FilePath:    "empty_hashes.txt",
			Chunks: []config.ChunkRef{
				{Hash: "", EncryptedHash: "", Size: 1024}, // Both hashes empty
				{Hash: "valid_chunk", Size: 2048},
			},
		}

		chunkRefs := map[string][]string{
			"valid_chunk": {"/empty/empty_hashes.txt", "/empty/other.txt"},
		}

		sharedChunks, savedBytes, _ := ComputeDedupStatsForFile(file, chunkRefs)

		// Only valid_chunk should be processed
		if sharedChunks != 1 {
			t.Errorf("Expected 1 shared chunk, got %d", sharedChunks)
		}

		expectedSavedBytes := int64(2048)
		if savedBytes != expectedSavedBytes {
			t.Errorf("Expected saved bytes %d, got %d", expectedSavedBytes, savedBytes)
		}
	})

	t.Run("FileWithDefaultChunkSize", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/default/",
			FilePath:    "default_size.txt",
			Chunks: []config.ChunkRef{
				{Hash: "default_chunk", Size: 0, EncryptedSize: 0}, // No size info
			},
		}

		chunkRefs := map[string][]string{
			"default_chunk": {"/default/default_size.txt", "/default/other.txt"},
		}

		sharedChunks, savedBytes, _ := ComputeDedupStatsForFile(file, chunkRefs)

		if sharedChunks != 1 {
			t.Errorf("Expected 1 shared chunk, got %d", sharedChunks)
		}

		// Should use default chunk size (4 MiB = 4 * 1024 * 1024)
		expectedSavedBytes := int64(4 * 1024 * 1024)
		if savedBytes != expectedSavedBytes {
			t.Errorf("Expected saved bytes %d, got %d", expectedSavedBytes, savedBytes)
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/empty/",
			FilePath:    "empty.txt",
			Chunks:      []config.ChunkRef{}, // No chunks
		}

		chunkRefs := map[string][]string{}

		sharedChunks, savedBytes, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		if sharedChunks != 0 {
			t.Errorf("Expected 0 shared chunks for empty file, got %d", sharedChunks)
		}

		if savedBytes != 0 {
			t.Errorf("Expected 0 saved bytes for empty file, got %d", savedBytes)
		}

		if len(sharedWith) != 0 {
			t.Errorf("Expected no shared files for empty file, got %v", sharedWith)
		}
	})

	t.Run("SharedWithDeterministicSorting", func(t *testing.T) {
		file := config.FileManifest{
			Destination: "/sort/",
			FilePath:    "test.txt",
			Chunks: []config.ChunkRef{
				{Hash: "shared_chunk", Size: 1024},
			},
		}

		// Provide files in non-alphabetical order
		chunkRefs := map[string][]string{
			"shared_chunk": {"/sort/test.txt", "/sort/zebra.txt", "/sort/alpha.txt", "/sort/beta.txt"},
		}

		sharedChunks, _, sharedWith := ComputeDedupStatsForFile(file, chunkRefs)

		if sharedChunks != 1 {
			t.Errorf("Expected 1 shared chunk, got %d", sharedChunks)
		}

		// Should be sorted alphabetically (excluding the file itself)
		expectedOrder := []string{"/sort/alpha.txt", "/sort/beta.txt", "/sort/zebra.txt"}
		if len(sharedWith) != len(expectedOrder) {
			t.Errorf("Expected %d shared files, got %d", len(expectedOrder), len(sharedWith))
		}

		for i, expected := range expectedOrder {
			if i < len(sharedWith) && sharedWith[i] != expected {
				t.Errorf("Expected file %s at position %d, got %s", expected, i, sharedWith[i])
			}
		}
	})
}