package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
	"gopkg.in/yaml.v3"
)

func WriteManifest(basePath string, config config.VaultConfig) error {
	manifestPath := filepath.Join(basePath, "vault.yaml")

	manifestFile, err := os.Create(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()
	encoder := yaml.NewEncoder(manifestFile)
	encoder.SetIndent(2)
	return encoder.Encode(config)
}

// StoreFileManifest saves a file manifest to the vault
func StoreFileManifest(vaultRoot string, fileName string, manifest *config.FileManifest) error {
	// Ensure manifests directory exists
	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifests directory: %v", err)
	}

	// Create manifest file path
	manifestPath := filepath.Join(manifestsDir, fileName+".yaml")

	// Create the file
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %v", err)
	}
	defer file.Close()

	// Encode the manifest to YAML
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %v", err)
	}

	return nil
}

// LoadFileManifest loads a file manifest from the vault
func LoadFileManifest(vaultRoot string, fileName string) (*config.FileManifest, error) {
	manifestPath := filepath.Join(vaultRoot, ".sietch", "manifests", fileName+".yaml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %v", err)
	}

	var manifest config.FileManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	return &manifest, nil
}

// ListFileManifests returns a list of all file manifests in the vault
func ListFileManifests(vaultRoot string) ([]string, error) {
	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")

	// Ensure manifests directory exists
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifests directory: %v", err)
	}

	// Extract manifest names (without .yaml extension)
	manifests := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			manifests = append(manifests, entry.Name()[:len(entry.Name())-5]) // Remove .yaml extension
		}
	}

	return manifests, nil
}
