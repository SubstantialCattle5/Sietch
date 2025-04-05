package config

import "time"

// VaultConfig represents the structure for vault.yaml
type VaultConfig struct {
	Name          string    `yaml:"name"`
	VaultID       string    `yaml:"vault_id"`
	CreatedAt     time.Time `yaml:"created_at"`
	SchemaVersion int       `yaml:"schema_version"`

	Encryption  EncryptionConfig `yaml:"encryption"`
	Chunking    ChunkingConfig   `yaml:"chunking"`
	Compression string           `yaml:"compression"`
	Sync        SyncConfig       `yaml:"sync"`
	Metadata    MetadataConfig   `yaml:"metadata"`
}

type EncryptionConfig struct {
	Type                string `yaml:"type"`
	KeyPath             string `yaml:"key_path"`
	PassphraseProtected bool   `yaml:"passphrase_protected"`
}

type ChunkingConfig struct {
	Strategy      string `yaml:"strategy"`
	ChunkSize     string `yaml:"chunk_size"`
	HashAlgorithm string `yaml:"hash_algorithm"`
}

type SyncConfig struct {
	Mode       string   `yaml:"mode"`
	KnownPeers []string `yaml:"known_peers,omitempty"`
}

type MetadataConfig struct {
	Author string   `yaml:"author"`
	Tags   []string `yaml:"tags"`
}

// BuildVaultConfig creates a complete vault configuration with all necessary fields
func BuildVaultConfig(
	vaultID, vaultName, author, keyType, keyPath string,
	passPhraseProtected bool,
	chunkingStrategy, chunkSize, hashAlgorithm, compression string,
	syncMode string,
	tags []string,
) VaultConfig {
	config := VaultConfig{
		VaultID:       vaultID,
		Name:          vaultName,
		CreatedAt:     time.Now().UTC(),
		SchemaVersion: 1, // Set schema version to 1 for initial version
		Compression:   compression,
	}

	// Set encryption configuration
	config.Encryption.Type = keyType
	config.Encryption.KeyPath = keyPath
	config.Encryption.PassphraseProtected = passPhraseProtected

	// Set chunking configuration
	config.Chunking.Strategy = chunkingStrategy
	config.Chunking.ChunkSize = chunkSize
	config.Chunking.HashAlgorithm = hashAlgorithm

	// Set sync configuration
	config.Sync.Mode = syncMode
	config.Sync.KnownPeers = []string{} // Initialize as empty array

	// Set metadata
	config.Metadata.Author = author
	config.Metadata.Tags = tags

	return config
}

// BuildDefaultVaultConfig creates a config with sensible defaults
func BuildDefaultVaultConfig(vaultID, vaultName, keyPath string) VaultConfig {
	return BuildVaultConfig(
		vaultID,
		vaultName,
		"nilay@dune.net", // Default author
		"aes",            // Default key type
		keyPath,
		false,    // Default no passphrase protection
		"fixed",  // Default chunking strategy
		"4MB",    // Default chunk size
		"sha256", // Default hash algorithm
		"none",   // Default compression
		"manual", // Default sync mode
		[]string{"research", "desert", "offline"}, // Default tags
	)
}
