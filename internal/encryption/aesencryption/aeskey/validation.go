package aeskey

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// GenerateKeyCheck creates a validation string to verify the key during decryption
func GenerateKeyCheck(key []byte) (string, error) {
	// Create a simple string that can be encrypted and later verified
	plaintext := []byte(constants.KeyValidationString)

	// Encrypt with the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Use GCM mode for the check
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	// Encrypt the validation string
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine nonce and ciphertext for storage
	result := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// VerifyPassphraseWithFallback verifies the passphrase using key check with fallback for legacy vaults
func VerifyPassphraseWithFallback(keyCheck string, derivedKey []byte) error {
	fmt.Printf("keys %v", derivedKey)
	err := VerifyPassphrase(keyCheck, derivedKey)
	if err != nil && strings.Contains(err.Error(), "key check too short") {
		// Try with legacy format (16-byte nonce)
		fmt.Println("Warning: Attempting fallback with 16-byte nonce (legacy format)")
		return VerifyLegacyPassphrase(keyCheck, derivedKey)
	}
	return err
}

// VerifyPassphrase verifies that the derived key is correct using the key check
func VerifyPassphrase(keyCheck string, derivedKey []byte) error {
	// Decode base64 key check
	checkData, err := base64.StdEncoding.DecodeString(keyCheck)
	if err != nil {
		return fmt.Errorf("invalid key check encoding: %w", err)
	}

	// Add diagnostics
	fmt.Printf("DEBUG: Key check length: %d bytes\n", len(checkData))

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("cipher creation failed: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("GCM mode creation failed: %w", err)
	}

	// The nonce size should be 12 bytes for GCM
	nonceSize := gcm.NonceSize()
	fmt.Printf("DEBUG: Expected nonce length: %d bytes\n", nonceSize)

	if len(checkData) < nonceSize {
		return fmt.Errorf("key check too short (%d bytes)", len(checkData))
	}

	// Extract nonce and ciphertext
	nonce := checkData[:nonceSize]
	ciphertext := checkData[nonceSize:]

	fmt.Printf("DEBUG: Extracted nonce (base64): %s\n",
		base64.StdEncoding.EncodeToString(nonce))
	fmt.Printf("DEBUG: Ciphertext length: %d bytes\n", len(ciphertext))

	// Try to decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("incorrect passphrase: key verification failed")
	}

	// Confirm expected string
	if string(plaintext) != constants.KeyValidationString {
		fmt.Printf("DEBUG: Got unexpected validation content: %s\n", string(plaintext))
		return fmt.Errorf("key validation failed: unexpected content")
	}

	fmt.Printf("DEBUG: Passphrase verification successful\n")
	return nil
}

// VerifyLegacyPassphrase verifies passphrase using legacy 16-byte nonce format
func VerifyLegacyPassphrase(keyCheck string, derivedKey []byte) error {
	// Decode base64 key check
	checkData, err := base64.StdEncoding.DecodeString(keyCheck)
	if err != nil {
		return fmt.Errorf("invalid key check encoding: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("cipher creation failed: %w", err)
	}

	// Force 16-byte nonce size for legacy vaults
	nonceSize := constants.LegacyNonceSize

	if len(checkData) < nonceSize {
		return fmt.Errorf("key check too short even for legacy format")
	}

	// Extract oversized nonce (16 bytes instead of 12)
	nonce := checkData[:nonceSize]
	ciphertext := checkData[nonceSize:]

	// Use only the first 12 bytes of the nonce for GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("GCM mode creation failed: %w", err)
	}

	// Try to decrypt with the first 12 bytes of the 16-byte nonce
	plaintext, err := gcm.Open(nil, nonce[:gcm.NonceSize()], ciphertext, nil)
	if err != nil {
		return fmt.Errorf("incorrect passphrase: key verification failed")
	}

	// Confirm expected string
	if string(plaintext) != constants.KeyValidationString {
		return fmt.Errorf("key validation failed: unexpected content")
	}

	fmt.Println("Warning: Successfully verified with legacy format. Consider reinitializing your vault.")
	return nil
}

// PrintKeyDetails provides diagnostic information about the vault key configuration
func PrintKeyDetails(cfg *config.VaultConfig) {
	fmt.Println("=== Vault Key Diagnostics ===")

	// Show AES config details
	if cfg.Encryption.AESConfig != nil {
		fmt.Printf("Mode: %s\n", cfg.Encryption.AESConfig.Mode)
		fmt.Printf("KDF: %s\n", cfg.Encryption.AESConfig.KDF)

		// Check nonce
		if nonceStr := cfg.Encryption.AESConfig.Nonce; nonceStr != "" {
			nonce, err := base64.StdEncoding.DecodeString(nonceStr)
			if err != nil {
				fmt.Printf("Nonce: [Invalid base64] %s\n", nonceStr)
			} else {
				fmt.Printf("Nonce: %d bytes (base64: %s)\n", len(nonce), nonceStr)
			}
		}

		// Check key check
		if keyCheck := cfg.Encryption.AESConfig.KeyCheck; keyCheck != "" {
			data, err := base64.StdEncoding.DecodeString(keyCheck)
			if err != nil {
				fmt.Printf("Key check: [Invalid base64] %s\n", keyCheck)
			} else {
				fmt.Printf("Key check: %d bytes\n", len(data))
			}
		}

		// Check key
		if key := cfg.Encryption.AESConfig.Key; key != "" {
			keyData, err := base64.StdEncoding.DecodeString(key)
			if err != nil {
				fmt.Printf("Key: [Invalid base64] %s\n", key)
			} else {
				fmt.Printf("Key: %d bytes\n", len(keyData))
				if len(keyData) > 12 {
					// Check if first 12 bytes match the nonce
					nonceStr := cfg.Encryption.AESConfig.Nonce
					if nonceStr != "" {
						nonce, _ := base64.StdEncoding.DecodeString(nonceStr)
						nonceLen := len(nonce)
						if nonceLen > 0 && bytes.Equal(keyData[:nonceLen], nonce) {
							fmt.Printf("WARNING: Key begins with the same bytes as the nonce!\n")
						}
					}
				}
			}
		}
	}
	fmt.Println("=============================")
}
