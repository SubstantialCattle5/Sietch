package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// Enhanced version of CreateVaultStructure
func CreateVaultStructure(basePath string) error {
	// Define the required directories
	dirs := []string{
		filepath.Join(basePath, ".sietch", "keys"),
		filepath.Join(basePath, ".sietch", "chunks"),
		filepath.Join(basePath, ".sietch", "state"),
		filepath.Join(basePath, ".sietch", "manifests"),
		filepath.Join(basePath, "data"),
	}

	// Create each directory with proper permissions
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create an empty state file
	statePath := filepath.Join(basePath, ".sietch", "state", "state.db")
	if _, err := os.Create(statePath); err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}

	return nil
}
