package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

func AesEncryption(data string, vaultConfig config.VaultConfig) (string, error) {
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

// AesEncryptWithPassphrase encrypts data using the vault's encryption key
// The passphrase is used to decrypt the encryption key if the vault is passphrase protected
func AesEncryptWithPassphrase(data string, vaultConfig config.VaultConfig, passphrase string) (string, error) {
	// Validate encryption type is AES
	if vaultConfig.Encryption.Type != constants.EncryptionTypeAES {
		return "", fmt.Errorf("vault is not configured for AES encryption (using %s)",
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

	// Create cipher block using the key
	block, err := aes.NewCipher(keyData)
	if err != nil {
		return "", fmt.Errorf("error creating AES cipher block: %w", err)
	}

	// Determine encryption mode from config or default to GCM
	mode := "gcm"
	if vaultConfig.Encryption.AESConfig != nil && vaultConfig.Encryption.AESConfig.Mode != "" {
		mode = vaultConfig.Encryption.AESConfig.Mode
	}

	if mode == "gcm" {
		// Use GCM mode for authenticated encryption
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", fmt.Errorf("error creating GCM: %w", err)
		}

		// Generate a random nonce
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return "", fmt.Errorf("error generating nonce: %w", err)
		}

		// Encrypt and authenticate the plaintext
		ciphertext := gcm.Seal(nonce, nonce, plainText, nil)

		// Return the encrypted data as a hex string
		return hex.EncodeToString(ciphertext), nil
	} else if mode == "cbc" {
		// Use CBC mode with PKCS#7 padding
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return "", fmt.Errorf("error generating IV: %w", err)
		}

		// Apply PKCS#7 padding
		padLength := aes.BlockSize - (len(plainText) % aes.BlockSize)
		padText := bytes.Repeat([]byte{byte(padLength)}, padLength)
		paddedData := append(plainText, padText...)

		// Create CBC encrypter
		cbcMode := cipher.NewCBCEncrypter(block, iv)
		ciphertext := make([]byte, len(paddedData))
		cbcMode.CryptBlocks(ciphertext, paddedData)

		// Prepend IV for storage
		result := append(iv, ciphertext...)

		return hex.EncodeToString(result), nil
	}

	return "", fmt.Errorf("unsupported encryption mode: %s", mode)
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

// loadEncryptionKeyWithPassphrase loads and decrypts the encryption key if needed
func loadEncryptionKeyWithPassphrase(keyPath string, passphrase string, encConfig config.EncryptionConfig) ([]byte, error) {
	// Read the key file
	encryptedKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}

	// If not passphrase protected, return the key as-is
	if !encConfig.PassphraseProtected {
		return encryptedKey, nil
	}

	// If passphrase protection is enabled but no passphrase provided
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase required for encrypted vault but not provided")
	}

	// Ensure AESConfig exists
	if encConfig.AESConfig == nil {
		return nil, fmt.Errorf("missing AES configuration for passphrase-protected key")
	}

	// Get salt from config
	salt, err := base64.StdEncoding.DecodeString(encConfig.AESConfig.Salt)
	if err != nil {
		return nil, fmt.Errorf("error decoding salt: %w", err)
	}

	// Derive key using appropriate KDF
	var derivedKey []byte
	if encConfig.AESConfig.KDF == "scrypt" {
		// Use scrypt KDF
		derivedKey, err = scrypt.Key(
			[]byte(passphrase),
			salt,
			encConfig.AESConfig.ScryptN,
			encConfig.AESConfig.ScryptR,
			encConfig.AESConfig.ScryptP,
			32, // 32 bytes for AES-256
		)
		if err != nil {
			return nil, fmt.Errorf("error deriving key with scrypt: %w", err)
		}
	} else if encConfig.AESConfig.KDF == "pbkdf2" {
		// Use PBKDF2 KDF
		derivedKey = pbkdf2.Key(
			[]byte(passphrase),
			salt,
			encConfig.AESConfig.PBKDF2I,
			32, // 32 bytes for AES-256
			sha256.New,
		)
	} else {
		return nil, fmt.Errorf("unsupported KDF algorithm: %s", encConfig.AESConfig.KDF)
	}

	// Verify the key using the key check value if available
	if encConfig.AESConfig.KeyCheck != "" {
		if !verifyKeyCheck(derivedKey, encConfig.AESConfig.KeyCheck) {
			return nil, fmt.Errorf("incorrect passphrase: key verification failed")
		}
	}

	// Decrypt the encryption key with the derived key
	decryptedKey, err := decryptKeyWithDerivedKey(encryptedKey, derivedKey, encConfig.AESConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return decryptedKey, nil
}

func verifyKeyCheck(derivedKey []byte, keyCheck string) bool {
	// Decode the key check value
	keyCheckData, err := base64.StdEncoding.DecodeString(keyCheck)
	if err != nil {
		return false
	}

	// Create cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return false
	}

	// Use GCM mode for verification
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return false
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(keyCheckData) < nonceSize {
		return false
	}
	nonce, ciphertext := keyCheckData[:nonceSize], keyCheckData[nonceSize:]

	// Try to decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return false
	}

	// Check if decrypted value matches expected validation string
	return string(plaintext) == "sietch-key-validation"
}

// decryptKeyWithDerivedKey decrypts the encryption key using the derived key
func decryptKeyWithDerivedKey(encryptedKey, derivedKey []byte, aesConfig *config.AESConfig) ([]byte, error) {
	// Create cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher block: %w", err)
	}

	// Get encryption mode
	mode := aesConfig.Mode
	if mode == "" {
		mode = "gcm" // Default mode is GCM
	}

	if mode == "gcm" {
		// Use GCM mode
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("error creating GCM: %w", err)
		}

		// Extract nonce
		nonceSize := gcm.NonceSize()
		if len(encryptedKey) < nonceSize {
			return nil, fmt.Errorf("encrypted key too short to contain nonce")
		}

		nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]

		// Decrypt
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("error decrypting key (incorrect passphrase?): %w", err)
		}

		return plaintext, nil
	} else if mode == "cbc" {
		// Use CBC mode
		ivSize := aes.BlockSize
		if len(encryptedKey) < ivSize {
			return nil, fmt.Errorf("encrypted key too short to contain IV")
		}

		iv, ciphertext := encryptedKey[:ivSize], encryptedKey[ivSize:]

		// Create CBC decrypter
		cbcMode := cipher.NewCBCDecrypter(block, iv)
		plaintext := make([]byte, len(ciphertext))
		cbcMode.CryptBlocks(plaintext, ciphertext)

		// Remove PKCS#7 padding
		paddingLen := int(plaintext[len(plaintext)-1])
		if paddingLen > len(plaintext) || paddingLen <= 0 {
			return nil, fmt.Errorf("invalid padding")
		}

		return plaintext[:len(plaintext)-paddingLen], nil
	}

	return nil, fmt.Errorf("unsupported encryption mode: %s", mode)
}
