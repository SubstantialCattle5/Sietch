package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/substantialcattle5/sietch/internal/config"
)

// WriteManifest writes the vault configuration to vault.yaml
// ensuring that any encryption keys in the config are properly stored
func WriteManifest(basePath string, config config.VaultConfig) error {
	manifestPath := filepath.Join(basePath, "vault.yaml")

	// Verify if the encryption key is present in the config
	if config.Encryption.Type == "aes" && config.Encryption.AESConfig != nil {
		// Log that we're storing a key in the manifest
		if config.Encryption.AESConfig.Key != "" {
			fmt.Println("Storing encryption key in vault configuration")
		} else {
			fmt.Println("Warning: No encryption key found in AESConfig")
		}
	}

	// Create manifest file with restricted permissions (0600) to secure the key
	// Only owner can read/write the file since it will contain sensitive key material
	manifestFile, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer manifestFile.Close()

	// Encode the config with proper indentation
	encoder := yaml.NewEncoder(manifestFile)
	encoder.SetIndent(2)

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode vault configuration: %w", err)
	}

	fmt.Printf("Vault configuration written to: %s\n", manifestPath)
	return nil
}

// StoreFileManifest saves a file manifest to the vault
func StoreFileManifest(vaultRoot string, fileName string, manifest *config.FileManifest) error {
	// Ensure manifests directory exists
	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
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

// LoadVaultConfig loads the vault configuration from vault.yaml
func LoadVaultConfig(vaultRoot string) (*config.VaultConfig, error) {
	manifestPath := filepath.Join(vaultRoot, "vault.yaml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault configuration: %w", err)
	}

	var config config.VaultConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse vault configuration: %w", err)
	}

	// Check if encryption key is present
	if config.Encryption.Type == "aes" && config.Encryption.AESConfig != nil {
		if config.Encryption.AESConfig.Key != "" {
			fmt.Println("Found encryption key in vault configuration")
		}
	}

	return &config, nil
}

func WriteKeyToFile(keyMaterial []byte, keyPath string) error {
	// Create directory structure for the key if it doesn't exist
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
	}

	// Write the key with secure permissions (only owner can read/write)
	if err := os.WriteFile(keyPath, keyMaterial, 0o600); err != nil {
		return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
	}

	fmt.Printf("Encryption key stored at: %s\n", keyPath)
	return nil
}
