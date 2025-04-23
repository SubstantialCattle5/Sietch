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
		return "", fmt.Errorf("vault is not configured for AES encryption (using %s)", vaultConfig.Encryption.Type)
	}
	fmt.Print(vaultConfig.Encryption.Type)

	// Load encryption key from the specified path
	keyData, err := loadEncryptionKey(vaultConfig.Encryption.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	plainText := []byte(data)
	key := make([]byte, 32)

	if _, err := rand.Reader.Read(keyData); err != nil {
		fmt.Println("error generating random encryption key ", err)
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("error creating aes block cipher", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println("error setting gcm mode", err)
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println("error generating the nonce ", err)
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plainText, nil)

	enc := hex.EncodeToString(ciphertext)
	fmt.Println("original data:", data)
	fmt.Println("encrypted data:", enc)
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

	// Here you might add logic to handle passphrase decryption if needed

	return keyData, nil
}
