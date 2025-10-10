package sneakernet

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	"gopkg.in/yaml.v3"
)

func TestSaveFileManifest(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sietch_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a SneakTransfer instance
	st := &SneakTransfer{}

	// Create a FileManifest with chunks
	fileManifest := config.FileManifest{
		FilePath:    "/test/file.txt",
		Size:        1024,
		ModTime:     "2023-01-01T00:00:00Z",
		Destination: "/test/file.txt",
		AddedAt:     time.Now(),
		Chunks: []config.ChunkRef{
			{
				Hash:          "chunk1hash",
				EncryptedHash: "encrypted1hash",
				Size:          512,
				EncryptedSize: 520,
				Index:         0,
				Deduplicated:  false,
			},
			{
				Hash:          "chunk2hash",
				EncryptedHash: "encrypted2hash",
				Size:          512,
				EncryptedSize: 520,
				Index:         1,
				Deduplicated:  true,
			},
		},
	}

	// Save the manifest
	manifestPath := filepath.Join(tempDir, "test_manifest.yaml")
	err = st.saveFileManifest(manifestPath, fileManifest)
	if err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	// Read the manifest back
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest file: %v", err)
	}

	// Unmarshal the YAML
	var loadedManifest config.FileManifest
	err = yaml.Unmarshal(data, &loadedManifest)
	if err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	// Verify the chunks are present
	if len(loadedManifest.Chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(loadedManifest.Chunks))
	}

	// Verify chunk details
	if loadedManifest.Chunks[0].Hash != "chunk1hash" {
		t.Errorf("Expected chunk 0 hash 'chunk1hash', got '%s'", loadedManifest.Chunks[0].Hash)
	}
	if loadedManifest.Chunks[0].EncryptedHash != "encrypted1hash" {
		t.Errorf("Expected chunk 0 encrypted hash 'encrypted1hash', got '%s'", loadedManifest.Chunks[0].EncryptedHash)
	}
	if loadedManifest.Chunks[0].Index != 0 {
		t.Errorf("Expected chunk 0 index 0, got %d", loadedManifest.Chunks[0].Index)
	}
	if loadedManifest.Chunks[0].Deduplicated != false {
		t.Errorf("Expected chunk 0 deduplicated false, got %t", loadedManifest.Chunks[0].Deduplicated)
	}

	if loadedManifest.Chunks[1].Hash != "chunk2hash" {
		t.Errorf("Expected chunk 1 hash 'chunk2hash', got '%s'", loadedManifest.Chunks[1].Hash)
	}
	if loadedManifest.Chunks[1].Deduplicated != true {
		t.Errorf("Expected chunk 1 deduplicated true, got %t", loadedManifest.Chunks[1].Deduplicated)
	}

	// Verify other fields
	if loadedManifest.FilePath != "/test/file.txt" {
		t.Errorf("Expected file path '/test/file.txt', got '%s'", loadedManifest.FilePath)
	}
	if loadedManifest.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", loadedManifest.Size)
	}
}
