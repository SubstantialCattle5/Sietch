package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"

	"gopkg.in/yaml.v3"
)

// StoreFileManifest saves a file manifest to the manifests directory
func StoreFileManifest(basePath string, filename string, manifest *config.FileManifest) error {
	manifestDir := filepath.Join(basePath, ".sietch", "manifests")
	manifestPath := filepath.Join(manifestDir, filename+".yaml")

	// Create the manifest file
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer file.Close()

	// Encode the manifest to YAML
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	return nil
}
