package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/substantialcattle5/sietch/internal/config"
)

func AesEncryption(data string, vaultPath string) (string, error) {
	// Load vault configuration
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is AES
	if vaultConfig.Encryption.Type != "aes" {
		return "", fmt.Errorf("vault is not configured for AES encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Load encryption key from the specified path
	keyData, err := loadEncryptionKey(vaultConfig.Encryption.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Ensure key is valid for AES (16, 24, or 32 bytes)
	if len(keyData) != 16 && len(keyData) != 24 && len(keyData) != 32 {
		return "", fmt.Errorf("invalid key length: %d bytes", len(keyData))
	}

	plainText := []byte(data)

	// Create cipher block using the loaded key data
	block, err := aes.NewCipher(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating AES cipher block: %w", err)
	}

	// Use GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error setting GCM mode: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error generating nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, plainText, nil)

	return hex.EncodeToString(ciphertext), nil
}

func AesDecryption(encryptedData string, vaultPath string) (string, error) {

	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is AES
	if vaultConfig.Encryption.Type != "aes" {
		return "", fmt.Errorf("vault is not configured for AES encryption (using %s)", vaultConfig.Encryption.Type)
	}

	// Load encryption key
	keyData, err := loadEncryptionKey(vaultConfig.Encryption.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Decode the hex encoded ciphertext
	decodedCipherText, err := hex.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("error decoding hex: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating AES cipher block: %w", err)
	}

	// Use GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error setting GCM mode: %w", err)
	}

	// Make sure the ciphertext is long enough to contain a nonce
	if len(decodedCipherText) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := decodedCipherText[:gcm.NonceSize()], decodedCipherText[gcm.NonceSize():]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("error decrypting data: %w", err)
	}

	return string(plaintext), nil
}

func loadEncryptionKey(keyPath string) ([]byte, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}
	return keyData, nil
}
