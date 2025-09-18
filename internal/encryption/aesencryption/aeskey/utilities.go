package aeskey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/constants"
)

// Random generation utilities

// generateNonce generates a random nonce for GCM mode
func generateNonce() (string, error) {
	nonce := make([]byte, constants.GCMNonceSize) // 12 bytes for GCM
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}

// generateIV generates a random IV for CBC mode
func generateIV() (string, error) {
	iv := make([]byte, constants.CBCIVSize) // 16 bytes for CBC
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(iv), nil
}

// generateSalt generates a random salt for key derivation
func generateSalt() (string, error) {
	salt := make([]byte, constants.SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// generateRandomKey generates a random key for AES-256
func generateRandomKey() ([]byte, error) {
	key := make([]byte, constants.AESKeySize) // 32 bytes for AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// File operations utilities

// backupKeyToFile creates a backup of the key at the specified path
func backupKeyToFile(key []byte, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, constants.SecureDirPerms); err != nil {
		return err
	}
	// Write file with restrictive permissions
	return os.WriteFile(path, key, constants.SecureFilePerms)
}

// expandPath expands ~ to home directory in file paths
func expandPath(path string) string {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// Cryptographic utilities

// calculateKeyHash computes SHA256 hash of the key for identification
func calculateKeyHash(key []byte) string {
	hash := sha256.Sum256(key)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// ensureKeyDirectoryExists creates the directory structure for key storage
func ensureKeyDirectoryExists(keyPath string) error {
	keyDir := filepath.Dir(keyPath)
	return os.MkdirAll(keyDir, constants.SecureDirPerms)
}

// writeKeyToFile writes the key material to file with secure permissions
func writeKeyToFile(keyMaterial []byte, keyPath string) error {
	if err := ensureKeyDirectoryExists(keyPath); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	if err := os.WriteFile(keyPath, keyMaterial, constants.SecureFilePerms); err != nil {
		return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
	}

	return nil
}

// validateKeySize checks if the key length is valid for AES
func validateKeySize(keyMaterial []byte) error {
	keyLen := len(keyMaterial)
	if keyLen != constants.AESKeySize128 && keyLen != constants.AESKeySize192 && keyLen != constants.AESKeySize {
		return fmt.Errorf("invalid key length %d bytes - must be %d, %d, or %d bytes for AES",
			keyLen, constants.AESKeySize128, constants.AESKeySize192, constants.AESKeySize)
	}
	return nil
}
