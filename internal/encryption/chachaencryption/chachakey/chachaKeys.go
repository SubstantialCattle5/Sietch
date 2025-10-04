package chachakey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// GenerateChaCha20Key creates a key configuration for ChaCha20 encryption
func GenerateChaCha20Key(cfg *config.VaultConfig, passphrase string) (*config.KeyConfig, error) {
	// Initialize key configuration
	keyConfig := &config.KeyConfig{
		ChaChaConfig: &config.ChaChaConfig{
			Mode: "poly1305",
		},
	}

	// Ensure ChaChaConfig exists in the vault configuration
	if cfg.Encryption.ChaChaConfig == nil {
		cfg.Encryption.ChaChaConfig = config.BuildDefaultChaChaConfig()
	}

	// Generate a 32-byte key for ChaCha20
	keyMaterial := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(rand.Reader, keyMaterial); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Calculate key hash for verification
	keyHash := sha256.Sum256(keyMaterial)
	keyConfig.KeyHash = base64.StdEncoding.EncodeToString(keyHash[:])

	// Process the key based on whether it's passphrase-protected or not
	if cfg.Encryption.PassphraseProtected {
		if passphrase == "" {
			return nil, fmt.Errorf("passphrase required for passphrase-protected keys")
		}

		// Generate salt for KDF
		salt := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}

		cfg.Encryption.ChaChaConfig.Salt = base64.StdEncoding.EncodeToString(salt)
		keyConfig.ChaChaConfig.Salt = cfg.Encryption.ChaChaConfig.Salt

		// Derive key from passphrase using scrypt or pbkdf2
		var derivedKey []byte
		var err error

		if cfg.Encryption.ChaChaConfig.KDF == constants.KDFScrypt {
			derivedKey, err = scrypt.Key(
				[]byte(passphrase),
				salt,
				cfg.Encryption.ChaChaConfig.ScryptN,
				cfg.Encryption.ChaChaConfig.ScryptR,
				cfg.Encryption.ChaChaConfig.ScryptP,
				chacha20poly1305.KeySize,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to derive key with scrypt: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unsupported KDF: %s (use scrypt)", cfg.Encryption.ChaChaConfig.KDF)
		}

		// Encrypt the key material with the derived key
		aead, err := chacha20poly1305.New(derivedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
		}

		nonce := make([]byte, aead.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}

		encryptedKey := aead.Seal(nonce, nonce, keyMaterial, nil)
		keyConfig.ChaChaConfig.Key = base64.StdEncoding.EncodeToString(encryptedKey)

		// Write the encrypted key to file
		if err := writeKeyToFile(cfg.Encryption.KeyPath, encryptedKey); err != nil {
			return nil, err
		}
	} else {
		// For unprotected keys, just encode and store
		keyConfig.ChaChaConfig.Key = base64.StdEncoding.EncodeToString(keyMaterial)

		// Write the raw key to file
		if err := writeKeyToFile(cfg.Encryption.KeyPath, keyMaterial); err != nil {
			return nil, err
		}
	}

	// Copy KDF parameters to keyConfig
	keyConfig.ChaChaConfig.KDF = cfg.Encryption.ChaChaConfig.KDF
	keyConfig.ChaChaConfig.ScryptN = cfg.Encryption.ChaChaConfig.ScryptN
	keyConfig.ChaChaConfig.ScryptR = cfg.Encryption.ChaChaConfig.ScryptR
	keyConfig.ChaChaConfig.ScryptP = cfg.Encryption.ChaChaConfig.ScryptP
	keyConfig.ChaChaConfig.Mode = cfg.Encryption.ChaChaConfig.Mode

	return keyConfig, nil
}

// writeKeyToFile writes the key material to a file with secure permissions
func writeKeyToFile(keyPath string, keyMaterial []byte) error {
	// Create directory structure for the key if it doesn't exist
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, constants.SecureDirPerms); err != nil {
		return fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
	}

	// Write the key with secure permissions (only owner can read/write)
	if err := os.WriteFile(keyPath, keyMaterial, constants.SecureFilePerms); err != nil {
		return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
	}

	return nil
}
