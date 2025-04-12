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

// EncryptionConfig contains encryption settings
type EncryptionConfig struct {
	Type                string     `yaml:"type"`
	KeyPath             string     `yaml:"key_path"`
	KeyHash             string     `yaml:"key_hash,omitempty"` // Fingerprint of the key
	PassphraseProtected bool       `yaml:"passphrase_protected"`
	KeyFile             bool       `yaml:"key_file,omitempty"`        // Whether key comes from file
	KeyFilePath         string     `yaml:"key_file_path,omitempty"`   // Path to key file
	RandomKey           bool       `yaml:"random_key,omitempty"`      // Whether key was randomly generated
	KeyBackupPath       string     `yaml:"key_backup_path,omitempty"` // Where key is backed up
	AESConfig           *AESConfig `yaml:"aes_config,omitempty"`      // AES specific settings
	GPGConfig           *GPGConfig `yaml:"gpg_config,omitempty"`      // GPG specific settings
}

// AESConfig contains AES-specific encryption settings
type AESConfig struct {
	Mode     string `yaml:"mode,omitempty"`      // GCM or CBC
	KDF      string `yaml:"kdf,omitempty"`       // scrypt or pbkdf2
	Salt     string `yaml:"salt,omitempty"`      // Base64 encoded salt
	ScryptN  int    `yaml:"scrypt_n,omitempty"`  // scrypt N parameter
	ScryptR  int    `yaml:"scrypt_r,omitempty"`  // scrypt r parameter
	ScryptP  int    `yaml:"scrypt_p,omitempty"`  // scrypt p parameter
	PBKDF2I  int    `yaml:"pbkdf2_i,omitempty"`  // PBKDF2 iterations
	Nonce    string `yaml:"nonce,omitempty"`     // For GCM/CTR modes
	IV       string `yaml:"iv,omitempty"`        // For CBC mode
	KeyCheck string `yaml:"key_check,omitempty"` // Hash to verify key
}

// GPGConfig contains GPG-specific encryption settings
type GPGConfig struct {
	KeyID      string `yaml:"key_id,omitempty"`      // GPG key ID
	Recipient  string `yaml:"recipient,omitempty"`   // Recipient for encryption
	PublicKey  string `yaml:"public_key,omitempty"`  // Path to public key
	PrivateKey string `yaml:"private_key,omitempty"` // Path to private key
	KeyServer  string `yaml:"key_server,omitempty"`  // Key server URL
}

// ChunkingConfig contains settings for file chunking
type ChunkingConfig struct {
	Strategy      string `yaml:"strategy"`
	ChunkSize     string `yaml:"chunk_size"`
	HashAlgorithm string `yaml:"hash_algorithm"`
}

// SyncConfig contains synchronization settings
type SyncConfig struct {
	Mode       string   `yaml:"mode"`
	KnownPeers []string `yaml:"known_peers,omitempty"`
}

// MetadataConfig contains user metadata
type MetadataConfig struct {
	Author string   `yaml:"author"`
	Tags   []string `yaml:"tags"`
}

// KeyConfig is the internal structure returned by key generation functions
type KeyConfig struct {
	KeyHash   string     `yaml:"key_hash,omitempty"`
	Salt      string     `yaml:"salt,omitempty"`
	AESConfig *AESConfig `yaml:"aes_config,omitempty"`
	GPGConfig *GPGConfig `yaml:"gpg_config,omitempty"`
}

// FileManifest represents the metadata for a stored file
type FileManifest struct {
	FilePath    string              `yaml:"file"`
	Size        int64               `yaml:"size"`
	ModTime     string              `yaml:"mtime"`
	Chunks      []ChunkRef          `yaml:"chunks"`
	Destination string              `yaml:"destination"`
	Tags        []string            `yaml:"tags,omitempty"`         // File-specific tags
	Encryption  *FileEncryptionInfo `yaml:"encryption,omitempty"`   // Per-file encryption settings
	ContentHash string              `yaml:"content_hash,omitempty"` // Hash of entire file content
	MerkleRoot  string              `yaml:"merkle_root,omitempty"`  // Root hash of chunk Merkle tree
	AddedAt     time.Time           `yaml:"added_at"`               // When file was added to vault
	LastSynced  time.Time           `yaml:"last_synced,omitempty"`  // Last successful sync time
}

// FileEncryptionInfo contains per-file encryption details (if different from vault default)
type FileEncryptionInfo struct {
	Type         string `yaml:"type,omitempty"`          // Can override vault encryption type
	KeyReference string `yaml:"key_reference,omitempty"` // References which key was used (vault_master or custom)
	IV           string `yaml:"iv,omitempty"`            // Initialization vector if applicable
	Nonce        string `yaml:"nonce,omitempty"`         // Nonce for GCM mode
}

// ChunkRef references a chunk in the vault
type ChunkRef struct {
	Hash          string `yaml:"hash"`                     // Hash of chunk content (pre-encryption)
	EncryptedHash string `yaml:"encrypted_hash,omitempty"` // Hash of encrypted chunk (filename in storage)
	Size          int64  `yaml:"size"`                     // Size of plaintext chunk
	EncryptedSize int64  `yaml:"encrypted_size,omitempty"` // Size after encryption
	Index         int    `yaml:"index"`                    // Position in the file
	Deduplicated  bool   `yaml:"deduplicated,omitempty"`   // Whether this chunk was deduplicated
	Compressed    bool   `yaml:"compressed,omitempty"`     // Whether this chunk was compressed
	IV            string `yaml:"iv,omitempty"`             // Per-chunk IV if used
	Integrity     string `yaml:"integrity,omitempty"`      // Integrity check value (e.g., HMAC)
}

// BuildVaultConfig creates a complete vault configuration with all necessary fields
func BuildVaultConfig(
	vaultID, vaultName, author, keyType, keyPath string,
	passPhraseProtected bool,
	chunkingStrategy, chunkSize, hashAlgorithm, compression string,
	syncMode string,
	tags []string,
	keyConfig ...*KeyConfig, // Optional key configuration
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

	// If key configuration is provided, apply it
	if len(keyConfig) > 0 && keyConfig[0] != nil {
		kc := keyConfig[0]
		config.Encryption.KeyHash = kc.KeyHash

		// Apply AES-specific config if available
		if kc.AESConfig != nil && keyType == "aes" {
			config.Encryption.AESConfig = kc.AESConfig
		}

		// Apply GPG-specific config if available
		if kc.GPGConfig != nil && keyType == "gpg" {
			config.Encryption.GPGConfig = kc.GPGConfig
		}
	}

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

// BuildDefaultAESConfig creates a default AES configuration
func BuildDefaultAESConfig() *AESConfig {
	return &AESConfig{
		Mode:    "gcm",
		KDF:     "scrypt",
		ScryptN: 32768,
		ScryptR: 8,
		ScryptP: 1,
	}
}

// BuildDefaultGPGConfig creates a default GPG configuration
func BuildDefaultGPGConfig() *GPGConfig {
	return &GPGConfig{
		KeyServer: "hkps://keys.openpgp.org",
	}
}
