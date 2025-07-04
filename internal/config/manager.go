package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Manager handles operations on a Sietch vault
type Manager struct {
	vaultRoot string
}

// Manifest represents the content of a vault
type Manifest struct {
	Files []FileManifest `json:"files"`
}

// NewManager creates a new vault manager
func NewManager(vaultRoot string) (*Manager, error) {
	return &Manager{
		vaultRoot: vaultRoot,
	}, nil
}

// GetManifest returns the vault manifest
func (m *Manager) GetManifest() (*Manifest, error) {
	manifestsDir := filepath.Join(m.vaultRoot, ".sietch", "manifests")
	manifest := &Manifest{
		Files: []FileManifest{},
	}

	// Ensure directory exists
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return manifest, nil // Return empty manifest if directory doesn't exist
	}

	// Read all manifest files
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return manifest, nil // Return empty manifest if error reading directory
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		// Load the file manifest
		filePath := filepath.Join(manifestsDir, entry.Name())
		fileManifest, err := loadFileManifest(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to load manifest %s: %v\n", entry.Name(), err)
			continue
		}

		manifest.Files = append(manifest.Files, *fileManifest)
	}

	return manifest, nil
}

// GetChunk retrieves a chunk by its hash
func (m *Manager) GetChunk(hash string) ([]byte, error) {
	chunkPath := filepath.Join(m.vaultRoot, ".sietch", "chunks", hash)
	fmt.Printf("chunk path %v\n", chunkPath) // Added newline here

	// Check if chunk exists
	if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chunk not found: %s", hash)
	}

	// Read the chunk data
	return os.ReadFile(chunkPath)
}

// StoreChunk stores a chunk in the vault
func (m *Manager) StoreChunk(hash string, data []byte) error {
	chunkPath := filepath.Join(m.vaultRoot, ".sietch", "chunks", hash)

	// Ensure chunks directory exists
	chunksDir := filepath.Join(m.vaultRoot, ".sietch", "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Write the chunk data
	return os.WriteFile(chunkPath, data, 0644)
}

// ChunkExists checks if a chunk exists in the vault
func (m *Manager) ChunkExists(hash string) (bool, error) {
	chunkPath := filepath.Join(m.vaultRoot, ".sietch", "chunks", hash)
	_, err := os.Stat(chunkPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// RebuildReferences rebuilds file references from manifests
func (m *Manager) RebuildReferences() error {
	// This is a placeholder for the actual implementation
	// In a real implementation, this would:
	// 1. Check all manifests for consistency
	// 2. Verify chunks exist for all file references
	// 3. Update any inconsistent references
	return nil
}

// Helper function to load a file manifest
func loadFileManifest(path string) (*FileManifest, error) {
	// Read manifest file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %v", err)
	}

	// Parse YAML content
	var manifest FileManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	return &manifest, nil
}

// GetConfig loads and returns the vault configuration
func (m *Manager) GetConfig() (*VaultConfig, error) {
	configPath := filepath.Join(m.vaultRoot, "vault.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("vault configuration not found: %v", err)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %v", err)
	}

	// Parse YAML content
	var config VaultConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}

	return &config, nil
}

// SaveConfig writes the vault configuration to disk
func (m *Manager) SaveConfig(config *VaultConfig) error {
	log.Printf("Saving vault configuration to %s", m.vaultRoot)
	configPath := filepath.Join(m.vaultRoot, "vault.yaml")

	// Ensure .sietch directory exists
	sietchDir := filepath.Join(m.vaultRoot, ".sietch")
	// log.Printf("Ensuring directory exists: %s", sietchDir)
	if err := os.MkdirAll(sietchDir, 0755); err != nil {
		// log.Printf("ERROR: Failed to create directory %s: %v", sietchDir, err)
		return fmt.Errorf("failed to create .sietch directory: %v", err)
	}
	// log.Printf("Directory verified: %s", sietchDir)

	// Marshal configuration to YAML
	// log.Printf("Marshaling configuration to YAML")
	data, err := yaml.Marshal(config)
	if err != nil {
		// log.Printf("ERROR: Failed to marshal configuration: %v", err)
		return fmt.Errorf("failed to marshal configuration: %v", err)
	}

	// // Pretty print full YAML to logs
	// log.Println("==== FULL CONFIG DUMP START ====")
	// log.Println(string(data))
	// log.Println("==== FULL CONFIG DUMP END ====")

	// Write to file
	// log.Printf("Writing configuration to %s", configPath)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Printf("ERROR: Failed to write configuration to %s: %v", configPath, err)
		return fmt.Errorf("failed to write configuration file: %v", err)
	}
	// log.Printf("Successfully saved vault configuration to %s", configPath)

	return nil
}

// SaveVaultConfig saves the vault configuration
// This is a shorthand for backward compatibility
func SaveVaultConfig(vaultRoot string, config *VaultConfig) error {
	log.Printf("SaveVaultConfig: Creating manager for vault root: %s", vaultRoot)

	manager, err := NewManager(vaultRoot)
	if err != nil {
		log.Printf("ERROR: Failed to create manager for %s: %v", vaultRoot, err)
		return err
	}

	log.Printf("SaveVaultConfig: Delegating to Manager.SaveConfig")
	return manager.SaveConfig(config)
}
