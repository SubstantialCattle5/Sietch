package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// creates the basic vault structure
func CreateVaultStructure(basePath string) error {
	// Define the required directories
	dirs := []string{
		filepath.Join(basePath, ".sietch", "keys"),
		filepath.Join(basePath, ".sietch", "chunks"),
		filepath.Join(basePath, ".sietch", "manifests"),
		filepath.Join(basePath, "data"),
	}

	// Create each directory with proper permissions
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
