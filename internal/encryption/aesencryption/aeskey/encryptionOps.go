package aeskey

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// EncryptionMode represents the encryption mode configuration
type EncryptionMode struct {
	Mode  string
	Nonce string // For GCM mode
	IV    string // For CBC mode
}

// SetupEncryptionMode initializes the encryption mode and generates necessary parameters
func SetupEncryptionMode(cfg *config.VaultConfig, keyConfig *config.KeyConfig) error {
	// Default to GCM if not specified
	if cfg.Encryption.AESConfig.Mode == "" {
		cfg.Encryption.AESConfig.Mode = constants.AESModeGCM
	}

	switch cfg.Encryption.AESConfig.Mode {
	case constants.AESModeGCM:
		return setupGCMMode(keyConfig)
	case constants.AESModeCBC:
		return setupCBCMode(keyConfig)
	default:
		return fmt.Errorf("unsupported AES mode: %s", cfg.Encryption.AESConfig.Mode)
	}
}

// setupGCMMode generates a nonce for GCM mode
func setupGCMMode(keyConfig *config.KeyConfig) error {
	nonce, err := generateNonce()
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	keyConfig.AESConfig.Nonce = nonce
	return nil
}

// setupCBCMode generates an IV for CBC mode
func setupCBCMode(keyConfig *config.KeyConfig) error {
	iv, err := generateIV()
	if err != nil {
		return fmt.Errorf("failed to generate IV: %w", err)
	}
	keyConfig.AESConfig.IV = iv
	return nil
}

/*
*
EncryptKeyWithDerivedKey encrypts the key material using the derived key
Encrypt the key material with the derived key
This is implementing a "key-wrapping" or "envelope encryption" pattern,
The key material is encrypted with the derived key, and the derived key is stored in the key configuration.
This is done to ensure that the key material is not stored in plain text in the key configuration.
*/
func EncryptKeyWithDerivedKey(keyMaterial, derivedKey []byte, aesConfig *config.AESConfig) ([]byte, error) {
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Normalize and validate mode
	mode := strings.TrimSpace(strings.ToLower(aesConfig.Mode))

	// Default to GCM mode if not specified
	if mode == "" {
		mode = constants.AESModeGCM
		aesConfig.Mode = mode
	}

	switch mode {
	case constants.AESModeGCM:
		return encryptWithGCM(keyMaterial, block, aesConfig)
	case constants.AESModeCBC:
		return encryptWithCBC(keyMaterial, block, aesConfig)
	default:
		return nil, fmt.Errorf("unsupported encryption mode: '%s' (must be 'gcm' or 'cbc')", mode)
	}
}

// encryptWithGCM encrypts data using AES-GCM mode
func encryptWithGCM(keyMaterial []byte, block cipher.Block, aesConfig *config.AESConfig) ([]byte, error) {
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Ensure nonce exists
	if aesConfig.Nonce == "" {
		nonce, err := generateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}
		aesConfig.Nonce = nonce
	}

	nonce, err := base64.StdEncoding.DecodeString(aesConfig.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Verify nonce length
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: got %d bytes, expected %d",
			len(nonce), gcm.NonceSize())
	}

	// Encrypt the key material
	encryptedKey := gcm.Seal(nil, nonce, keyMaterial, nil)

	// Prepend nonce for storage
	return append(nonce, encryptedKey...), nil
}

// encryptWithCBC encrypts data using AES-CBC mode
func encryptWithCBC(keyMaterial []byte, block cipher.Block, aesConfig *config.AESConfig) ([]byte, error) {
	if aesConfig.IV == "" {
		iv, err := generateIV()
		if err != nil {
			return nil, fmt.Errorf("failed to generate IV: %w", err)
		}
		aesConfig.IV = iv
	}

	iv, err := base64.StdEncoding.DecodeString(aesConfig.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("IV length must be %d bytes for CBC mode", aes.BlockSize)
	}

	// Implement PKCS#7 padding
	padLength := aes.BlockSize - (len(keyMaterial) % aes.BlockSize)
	padText := bytes.Repeat([]byte{byte(padLength)}, padLength)
	paddedData := append(keyMaterial, padText...)

	// Create CBC encrypter
	cbc := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(paddedData))
	cbc.CryptBlocks(ciphertext, paddedData)

	// Prepend IV for storage
	return append(iv, ciphertext...), nil
}

// DecryptWithGCM decrypts data using AES-GCM mode
func DecryptWithGCM(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short: got %d bytes, need at least %d for nonce",
			len(data), nonceSize)
	}

	fmt.Printf("Debug: Extracting %d-byte nonce from %d-byte data\n", nonceSize, len(data))
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// DecryptWithCBC decrypts data using AES-CBC mode
func DecryptWithCBC(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	cbc := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	cbc.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS#7 padding
	return removePKCS7Padding(plaintext)
}

// removePKCS7Padding removes PKCS#7 padding from plaintext
func removePKCS7Padding(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}

	padLen := int(plaintext[len(plaintext)-1])
	if padLen > aes.BlockSize || padLen > len(plaintext) {
		return nil, fmt.Errorf("invalid padding")
	}

	// Validate padding
	for i := len(plaintext) - padLen; i < len(plaintext); i++ {
		if plaintext[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return plaintext[:len(plaintext)-padLen], nil
}
