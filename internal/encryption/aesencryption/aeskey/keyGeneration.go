package aeskey

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/substantialcattle5/sietch/internal/config"
)

// KeyGenerationOptions encapsulates options for key generation
type KeyGenerationOptions struct {
	PassphraseProtected bool
	UseKeyFile          bool
	KeyFilePath         string
	KeyPath             string
	KeyBackupPath       string
}

// GenerateKeyMaterial generates the raw key material based on configuration
func GenerateKeyMaterial(opts KeyGenerationOptions) ([]byte, error) {
	if opts.UseKeyFile && opts.KeyFilePath != "" {
		return loadKeyFromFile(opts.KeyFilePath)
	}
	return generateRandomKey()
}

// loadKeyFromFile reads and validates a key from file
func loadKeyFromFile(keyFilePath string) ([]byte, error) {
	expandedPath := expandPath(keyFilePath)
	keyMaterial, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", expandedPath, err)
	}

	// Validate the key size
	if err := validateKeySize(keyMaterial); err != nil {
		return nil, err
	}

	return keyMaterial, nil
}

// ProcessPassphraseProtectedKey handles the creation of passphrase-protected keys
func ProcessPassphraseProtectedKey(cfg *config.VaultConfig, keyConfig *config.KeyConfig, passphrase string, keyMaterial []byte) error {
	// Generate salt for key derivation
	salt, err := generateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	keyConfig.Salt = salt
	keyConfig.AESConfig.Salt = salt

	// Setup KDF defaults and copy parameters
	SetupKDFDefaults(cfg)
	CopyKDFParametersToKeyConfig(cfg, keyConfig)

	// Build KDF configuration
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return fmt.Errorf("failed to decode salt: %w", err)
	}

	kdfConfig := BuildKDFConfig(cfg, saltBytes)

	// Derive key from passphrase
	derivedKey, err := DeriveKey(passphrase, kdfConfig)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	encryptedKeyMaterial, err := EncryptKeyWithDerivedKey(keyMaterial, derivedKey, keyConfig.AESConfig)
	if err != nil {
		return fmt.Errorf("failed to encrypt key material: %w", err)
	}

	// Store the encrypted key material and its hash
	encodedKey := base64.StdEncoding.EncodeToString(encryptedKeyMaterial)
	keyConfig.AESConfig.Key = encodedKey
	keyConfig.KeyHash = calculateKeyHash(encryptedKeyMaterial)

	// Generate key check for validation during decryption
	keyCheck, err := GenerateKeyCheck(derivedKey)
	if err != nil {
		return fmt.Errorf("failed to generate key check: %w", err)
	}
	keyConfig.AESConfig.KeyCheck = keyCheck

	return nil
}

// ProcessUnprotectedKey handles the creation of unprotected keys
func ProcessUnprotectedKey(keyConfig *config.KeyConfig, keyMaterial []byte) {
	encodedKey := base64.StdEncoding.EncodeToString(keyMaterial)
	keyConfig.AESConfig.Key = encodedKey
	keyConfig.KeyHash = calculateKeyHash(keyMaterial)
}

// HandleKeyStorage manages writing keys to files and creating backups
func HandleKeyStorage(keyMaterial []byte, opts KeyGenerationOptions) error {
	// Write main key file if KeyPath is specified (always write for vaults)
	if opts.KeyPath != "" {
		if err := writeKeyToFile(keyMaterial, opts.KeyPath); err != nil {
			return err
		}
		fmt.Printf("Encryption key stored at: %s\n", opts.KeyPath)
	}

	// Create backup if requested
	if opts.KeyBackupPath != "" {
		expandedBackupPath := expandPath(opts.KeyBackupPath)
		if err := backupKeyToFile(keyMaterial, expandedBackupPath); err != nil {
			return fmt.Errorf("failed to backup key to %s: %w", expandedBackupPath, err)
		}
		fmt.Printf("Key backed up to: %s\n", expandedBackupPath)
	}

	return nil
}

// InitializeKeyConfig creates and initializes a new KeyConfig
func InitializeKeyConfig() *config.KeyConfig {
	return &config.KeyConfig{
		AESConfig: &config.AESConfig{},
	}
}

// EnsureAESConfig ensures that AESConfig exists in the vault configuration
func EnsureAESConfig(cfg *config.VaultConfig) {
	if cfg.Encryption.AESConfig == nil {
		cfg.Encryption.AESConfig = config.BuildDefaultAESConfig()
	}
}

// SyncConfigKeys synchronizes key data between vault and key configurations
func SyncConfigKeys(vaultCfg *config.VaultConfig, keyCfg *config.KeyConfig) {
	if vaultCfg.Encryption.AESConfig != nil {
		vaultCfg.Encryption.AESConfig.Key = keyCfg.AESConfig.Key
	}
}

// BuildKeyGenerationOptions creates options from vault configuration
func BuildKeyGenerationOptions(cfg *config.VaultConfig) KeyGenerationOptions {
	return KeyGenerationOptions{
		PassphraseProtected: cfg.Encryption.PassphraseProtected,
		UseKeyFile:          cfg.Encryption.KeyFile,
		KeyFilePath:         cfg.Encryption.KeyFilePath,
		KeyPath:             cfg.Encryption.KeyPath,
		KeyBackupPath:       cfg.Encryption.KeyBackupPath,
	}
}
