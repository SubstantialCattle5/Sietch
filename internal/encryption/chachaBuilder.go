package encryption

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// ChaCha20Encryption encrypts data using ChaCha20-Poly1305
func ChaCha20Encryption(data string, vaultConfig config.VaultConfig) (string, error) {
	// Validate encryption type is ChaCha20
	if vaultConfig.Encryption.Type != constants.EncryptionTypeChaCha20 {
		return "", fmt.Errorf("vault is not configured for ChaCha20 encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Load encryption key from the specified path
	keyData, err := loadEncryptionKey(vaultConfig.Encryption.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Ensure key is valid for ChaCha20 (32 bytes)
	if len(keyData) != chacha20poly1305.KeySize {
		return "", fmt.Errorf("invalid key length: %d bytes (ChaCha20 requires %d bytes)", len(keyData), chacha20poly1305.KeySize)
	}

	plainText := []byte(data)

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating ChaCha20-Poly1305 cipher: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error generating nonce: %w", err)
	}

	// Encrypt data
	ciphertext := aead.Seal(nonce, nonce, plainText, nil)

	return hex.EncodeToString(ciphertext), nil
}

// ChaCha20EncryptWithPassphrase encrypts data using ChaCha20-Poly1305 with passphrase
func ChaCha20EncryptWithPassphrase(data string, vaultConfig config.VaultConfig, passphrase string) (string, error) {
	// Validate encryption type is ChaCha20
	if vaultConfig.Encryption.Type != constants.EncryptionTypeChaCha20 {
		return "", fmt.Errorf("vault is not configured for ChaCha20 encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Load and decrypt the encryption key if necessary
	keyData, err := loadEncryptionKeyWithPassphrase(
		vaultConfig.Encryption.KeyPath,
		passphrase,
		vaultConfig.Encryption,
	)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	plainText := []byte(data)

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating ChaCha20-Poly1305 cipher: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error generating nonce: %w", err)
	}

	// Encrypt data
	ciphertext := aead.Seal(nonce, nonce, plainText, nil)

	return hex.EncodeToString(ciphertext), nil
}

// ChaCha20Decryption decrypts data using ChaCha20-Poly1305
func ChaCha20Decryption(encryptedData string, vaultPath string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is ChaCha20
	if vaultConfig.Encryption.Type != constants.EncryptionTypeChaCha20 {
		return "", fmt.Errorf("vault is not configured for ChaCha20 encryption (using %s)", vaultConfig.Encryption.Type)
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

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating ChaCha20-Poly1305 cipher: %w", err)
	}

	// Make sure the ciphertext is long enough to contain a nonce
	if len(decodedCipherText) < aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := decodedCipherText[:aead.NonceSize()], decodedCipherText[aead.NonceSize():]

	// Decrypt the data
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("error decrypting data: %w", err)
	}

	return string(plaintext), nil
}

// ChaCha20DecryptionWithPassphrase decrypts data using ChaCha20-Poly1305 with passphrase
func ChaCha20DecryptionWithPassphrase(encryptedData string, vaultPath string, passphrase string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is ChaCha20
	if vaultConfig.Encryption.Type != constants.EncryptionTypeChaCha20 {
		return "", fmt.Errorf("vault is not configured for ChaCha20 encryption (using %s)", vaultConfig.Encryption.Type)
	}

	// Load and decrypt the encryption key if necessary
	keyData, err := loadEncryptionKeyWithPassphrase(
		vaultConfig.Encryption.KeyPath,
		passphrase,
		vaultConfig.Encryption,
	)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Decode the hex encoded ciphertext
	decodedCipherText, err := hex.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("error decoding hex: %w", err)
	}

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating ChaCha20-Poly1305 cipher: %w", err)
	}

	// Make sure the ciphertext is long enough to contain a nonce
	if len(decodedCipherText) < aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := decodedCipherText[:aead.NonceSize()], decodedCipherText[aead.NonceSize():]

	// Decrypt the data
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("error decrypting data: %w", err)
	}

	return string(plaintext), nil
}
