package aeskey

import (
	"encoding/base64"
	"fmt"

	"github.com/substantialcattle5/sietch/internal/config"
)

// GenerateAESKey creates a key configuration based on vault settings
// and optionally stores the key in memory rather than writing to file
func GenerateAESKey(cfg *config.VaultConfig, passphrase string) (*config.KeyConfig, error) {
	fmt.Printf("Vault Configuration: %+v\n", cfg)

	// Initialize key configuration
	keyConfig := InitializeKeyConfig()

	// Ensure AESConfig exists in the vault configuration
	EnsureAESConfig(cfg)

	// Setup encryption mode (GCM/CBC) and generate necessary parameters
	if err := SetupEncryptionMode(cfg, keyConfig); err != nil {
		return nil, err
	}

	// Build key generation options from configuration
	opts := BuildKeyGenerationOptions(cfg)

	// Generate the raw key material
	keyMaterial, err := GenerateKeyMaterial(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key material: %w", err)
	}

	// Process the key based on whether it's passphrase-protected or not
	if cfg.Encryption.PassphraseProtected {
		if err := ProcessPassphraseProtectedKey(cfg, keyConfig, passphrase, keyMaterial); err != nil {
			return nil, err
		}
		// For passphrase-protected keys, we need the encrypted key material for storage
		encryptedKey, err := base64.StdEncoding.DecodeString(keyConfig.AESConfig.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode encrypted key: %w", err)
		}
		keyMaterial = encryptedKey
	} else {
		ProcessUnprotectedKey(keyConfig, keyMaterial)
	}

	// Synchronize key data between configurations
	SyncConfigKeys(cfg, keyConfig)

	// Handle key storage (files and backups)
	if err := HandleKeyStorage(keyMaterial, opts); err != nil {
		return nil, err
	}

	return keyConfig, nil
}

// LoadEncryptionKey loads the encryption key using the provided passphrase
func LoadEncryptionKey(cfg *config.VaultConfig, passphrase string) ([]byte, error) {
	// PrintKeyDetails(cfg) // Debug info about the key configuration

	// Extract key check and salt from config
	keyCheck := cfg.Encryption.AESConfig.KeyCheck
	salt := cfg.Encryption.AESConfig.Salt

	// Add debug logging
	fmt.Printf("Debug: Loading encryption key with KDF=%s\n", cfg.Encryption.AESConfig.KDF)

	// Build KDF configuration and derive key from passphrase
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	kdfConfig := BuildKDFConfig(cfg, saltBytes)
	derivedKey, err := DeriveKey(passphrase, kdfConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Verify passphrase using key check with fallback for legacy vaults
	if err := VerifyPassphraseWithFallback(keyCheck, derivedKey); err != nil {
		return nil, fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Load and decrypt the key
	encryptedKey, err := base64.StdEncoding.DecodeString(cfg.Encryption.AESConfig.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	// Decrypt the key using GCM mode
	key, err := DecryptWithGCM(encryptedKey, derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return key, nil
}
