package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateVaultStructure(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, ".sietch", "keys"),
		filepath.Join(basePath, ".sietch", "chunks"),
		filepath.Join(basePath, ".sietch", "state"),
		filepath.Join(basePath, "data"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
