package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateKey(keyType string, keyPath string, usePassphrase bool) error {

	// create a directory
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	switch keyType {
	case "aes":
		return generateAESKey(keyPath, usePassphrase)
	case "gpg":
		return generateGPGKey(keyPath)
	case "none":
		return nil
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// todo : fcking fix it bullshit method for now

func generateGPGKey(keyPath string) error {
	// Simulate GPG key generation
	// In a real implementation, this might call out to the gpg command-line tool
	gpgKey := []byte("-----BEGIN PGP PUBLIC KEY-----\nExampleGPGKeyData\n-----END PGP PUBLIC KEY-----")

	// Write the GPG key to the specified file path
	if err := os.WriteFile(keyPath, gpgKey, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

func generateAESKey(keyPath string, usePassphrase bool) error {
	// Generate a random 256-bit AES key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate random key: %w", err)
	}

	// If passphrase protection is enabled, simulate passphrase encryption
	if usePassphrase {
		// In a real implementation, we would encrypt the key with the passphrase
		// For now, just encode the key as base64 and add a marker
		encodedKey := base64.StdEncoding.EncodeToString(key)
		keyWithPassphrase := []byte("PASSPHRASE_PROTECTED:" + encodedKey)

		// Write the key to the specified file path
		if err := os.WriteFile(keyPath, keyWithPassphrase, 0600); err != nil {
			return fmt.Errorf("failed to write key file: %w", err)
		}
	} else {
		// Write the raw key to the specified file path
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return fmt.Errorf("failed to write key file: %w", err)
		}
	}

	return nil
}
