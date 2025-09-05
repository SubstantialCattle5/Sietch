package keys

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// Cryptographic constants
const (
	// Key sizes in bytes
	AESKeySize    = 32 // AES-256 key size
	AESKeySize128 = 16 // AES-128 key size
	AESKeySize192 = 24 // AES-192 key size

	// Nonce and IV sizes
	GCMNonceSize    = 12 // Standard GCM nonce size
	CBCIVSize       = 16 // CBC initialization vector size
	SaltSize        = 16 // Salt size for key derivation
	LegacyNonceSize = 16 // Legacy nonce size for backward compatibility

	// Default KDF parameters
	DefaultScryptN     = 32768 // CPU/memory cost parameter
	DefaultScryptR     = 8     // Block size parameter
	DefaultScryptP     = 1     // Parallelization parameter
	DefaultPBKDF2Iters = 10000 // Default PBKDF2 iteration count

	// File permissions
	SecureDirPerms  = 0o700 // Owner read/write/execute only
	SecureFilePerms = 0o600 // Owner read/write only

	// Validation string for key verification
	KeyValidationString = "sietch-key-validation"
)

// GenerateAESKey creates a key configuration based on vault settings
// and optionally stores the key in memory rather than writing to file
func GenerateAESKey(cfg *config.VaultConfig, passphrase string) (*config.KeyConfig, error) {
	fmt.Printf("Vault Configuration: %+v\n", cfg)

	// Initialize key configuration with empty AESConfig if not present
	keyConfig := &config.KeyConfig{
		AESConfig: &config.AESConfig{},
	}

	// Ensure AESConfig exists in the vault configuration
	if cfg.Encryption.AESConfig == nil {
		cfg.Encryption.AESConfig = config.BuildDefaultAESConfig()
	}

	// gcm vs cbc question - https://security.stackexchange.com/questions/184305/why-would-i-ever-use-aes-256-cbc-if-aes-256-gcm-is-more-secure

	// Generate nonce/IV based on the selected encryption mode
	switch cfg.Encryption.AESConfig.Mode {
	case constants.AESModeGCM, "":
		nonce, err := generateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}
		keyConfig.AESConfig.Nonce = nonce
		// Set default mode to GCM if not specified
		if cfg.Encryption.AESConfig.Mode == "" {
			cfg.Encryption.AESConfig.Mode = "gcm"
		}
	case "cbc":
		iv, err := generateIV()
		if err != nil {
			return nil, fmt.Errorf("failed to generate IV: %w", err)
		}
		keyConfig.AESConfig.IV = iv
	default:
		return nil, fmt.Errorf("unsupported AES mode: %s", cfg.Encryption.AESConfig.Mode)
	}

	var keyMaterial []byte
	var err error

	// Handle passphrase-protected encryption
	if cfg.Encryption.PassphraseProtected {
		// Generate salt for key derivation
		salt, err := generateSalt()
		if err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
		keyConfig.Salt = salt
		keyConfig.AESConfig.Salt = salt

		// Generate a random key that will be encrypted with the passphrase
		keyMaterial, err = generateRandomKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate random key material: %w", err)
		}

		// Set KDF algorithm type from config or default to scrypt
		if cfg.Encryption.AESConfig.KDF == "" {
			cfg.Encryption.AESConfig.KDF = constants.KDFScrypt
		}
		keyConfig.AESConfig.KDF = cfg.Encryption.AESConfig.KDF

		var derivedKey []byte

		// Generate key using selected KDF
		switch cfg.Encryption.AESConfig.KDF {
		case "scrypt":
			// Set default scrypt parameters if not specified
			if cfg.Encryption.AESConfig.ScryptN == 0 {
				cfg.Encryption.AESConfig.ScryptN = DefaultScryptN
			}
			if cfg.Encryption.AESConfig.ScryptR == 0 {
				cfg.Encryption.AESConfig.ScryptR = DefaultScryptR
			}
			if cfg.Encryption.AESConfig.ScryptP == 0 {
				cfg.Encryption.AESConfig.ScryptP = DefaultScryptP
			}

			// Copy parameters to key config
			keyConfig.AESConfig.ScryptN = cfg.Encryption.AESConfig.ScryptN
			keyConfig.AESConfig.ScryptR = cfg.Encryption.AESConfig.ScryptR
			keyConfig.AESConfig.ScryptP = cfg.Encryption.AESConfig.ScryptP

			// Generate derived key using scrypt
			derivedKey, err = scrypt.Key(
				[]byte(passphrase),
				[]byte(salt),
				cfg.Encryption.AESConfig.ScryptN,
				cfg.Encryption.AESConfig.ScryptR,
				cfg.Encryption.AESConfig.ScryptP,
				AESKeySize, // 32 bytes for AES-256
			)
			if err != nil {
				return nil, fmt.Errorf("failed to derive key using scrypt: %w", err)
			}
		case "pbkdf2":
			// Set default PBKDF2 iterations if not specified
			if cfg.Encryption.AESConfig.PBKDF2I == 0 {
				cfg.Encryption.AESConfig.PBKDF2I = DefaultPBKDF2Iters
			}

			keyConfig.AESConfig.PBKDF2I = cfg.Encryption.AESConfig.PBKDF2I

			// Generate key using PBKDF2
			derivedKey = pbkdf2.Key(
				[]byte(passphrase),
				[]byte(salt),
				cfg.Encryption.AESConfig.PBKDF2I,
				AESKeySize, // 32 bytes for AES-256
				sha256.New,
			)
		default:
			return nil, fmt.Errorf("unsupported KDF algorithm: %s", cfg.Encryption.AESConfig.KDF)
		}

		// Encrypt the key material with the derived key
		encryptedKeyMaterial, err := encryptKeyWithDerivedKey(keyMaterial, derivedKey, keyConfig.AESConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt key material: %w", err)
		}

		// Store the encrypted key material and its hash
		keyMaterial = encryptedKeyMaterial
		keyConfig.KeyHash = calculateKeyHash(keyMaterial)

		// Add a key check for validation during decryption
		keyCheck, err := generateKeyCheck(derivedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key check: %w", err)
		}
		keyConfig.AESConfig.KeyCheck = keyCheck
	} else {
		// Handle key file or random key generation
		if cfg.Encryption.KeyFile {
			// Use existing key file
			expandedPath := expandPath(cfg.Encryption.KeyFilePath)
			keyMaterial, err = os.ReadFile(expandedPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read key file %s: %w", expandedPath, err)
			}

			// Verify the key is valid for AES (must be 16, 24, or 32 bytes)
			keyLen := len(keyMaterial)
			if keyLen != AESKeySize128 && keyLen != AESKeySize192 && keyLen != AESKeySize {
				return nil, fmt.Errorf("invalid key length %d bytes - must be %d, %d, or %d bytes for AES", keyLen, AESKeySize128, AESKeySize192, AESKeySize)
			}
		} else {
			// Generate random key
			keyMaterial, err = generateRandomKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate random key: %w", err)
			}
		}

		keyConfig.KeyHash = calculateKeyHash(keyMaterial)
	}

	// Store the key in the struct using base64 encoding
	encodedKey := base64.StdEncoding.EncodeToString(keyMaterial)
	keyConfig.AESConfig.Key = encodedKey

	// Also store it in the original config struct
	if cfg.Encryption.AESConfig != nil {
		cfg.Encryption.AESConfig.Key = encodedKey
	}

	// Optionally write to file if requested
	if cfg.Encryption.KeyPath != "" && cfg.Encryption.KeyFile {
		// Create directory structure for the key if it doesn't exist
		keyDir := filepath.Dir(cfg.Encryption.KeyPath)
		if err := os.MkdirAll(keyDir, SecureDirPerms); err != nil {
			return nil, fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
		}

		// Write the key with secure permissions
		if err := os.WriteFile(cfg.Encryption.KeyPath, keyMaterial, SecureFilePerms); err != nil {
			return nil, fmt.Errorf("failed to write key to %s: %w", cfg.Encryption.KeyPath, err)
		}

		fmt.Printf("Encryption key stored at: %s\n", cfg.Encryption.KeyPath)
	}

	// Backup key if requested
	if cfg.Encryption.KeyBackupPath != "" {
		expandedBackupPath := expandPath(cfg.Encryption.KeyBackupPath)
		if err := backupKeyToFile(keyMaterial, expandedBackupPath); err != nil {
			return nil, fmt.Errorf("failed to backup key to %s: %w", expandedBackupPath, err)
		}
		fmt.Printf("Key backed up to: %s\n", expandedBackupPath)
	}

	return keyConfig, nil
}

// LoadEncryptionKey loads the encryption key using the provided passphrase
func LoadEncryptionKey(cfg *config.VaultConfig, passphrase string) ([]byte, error) {
	printKeyDetails(cfg) // Debug info about the key configuration

	// Extract key check and salt from config
	keyCheck := cfg.Encryption.AESConfig.KeyCheck
	salt := cfg.Encryption.AESConfig.Salt

	// Add debug logging
	fmt.Printf("Debug: Loading encryption key with KDF=%s\n", cfg.Encryption.AESConfig.KDF)

	// Derive key from passphrase using the appropriate KDF
	var derivedKey []byte
	var err error

	switch cfg.Encryption.AESConfig.KDF {
	case "scrypt":
		derivedKey, err = scrypt.Key(
			[]byte(passphrase),
			[]byte(salt),
			cfg.Encryption.AESConfig.ScryptN,
			cfg.Encryption.AESConfig.ScryptR,
			cfg.Encryption.AESConfig.ScryptP,
			AESKeySize, // 32 bytes for AES-256
		)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key using scrypt: %w", err)
		}
	case "pbkdf2":
		derivedKey = pbkdf2.Key(
			[]byte(passphrase),
			[]byte(salt),
			cfg.Encryption.AESConfig.PBKDF2I,
			AESKeySize, // 32 bytes for AES-256
			sha256.New,
		)
	default:
		return nil, fmt.Errorf("unsupported KDF algorithm: %s", cfg.Encryption.AESConfig.KDF)
	}

	// Verify passphrase using key check with fallback for legacy vaults
	if err := verifyPassphraseWithFallback(keyCheck, derivedKey); err != nil {
		return nil, fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Load and decrypt the key
	encryptedKey, err := base64.StdEncoding.DecodeString(cfg.Encryption.AESConfig.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	// Directly use decryptWithGCM which properly extracts the prepended nonce
	key, err := decryptWithGCM(encryptedKey, derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return key, nil
}

// encryptKeyWithDerivedKey encrypts the key material using the derived key
func encryptKeyWithDerivedKey(keyMaterial, derivedKey []byte, aesConfig *config.AESConfig) ([]byte, error) {
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Normalize and validate mode
	mode := strings.TrimSpace(strings.ToLower(aesConfig.Mode))

	// Default to GCM mode if not specified
	if mode == "" {
		mode = "gcm"
		// Update the config to match our selection
		aesConfig.Mode = mode
	}

	switch mode {
	case "gcm":
		// For GCM mode
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

	case "cbc":
		// CBC mode implementation (unchanged)
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

	default:
		return nil, fmt.Errorf("unsupported encryption mode: '%s' (must be 'gcm' or 'cbc')", mode)
	}
}

// Improved error handling and diagnostics
func decryptWithGCM(data, key []byte) ([]byte, error) {
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

func decryptWithCBC(data, key []byte) ([]byte, error) {
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

func verifyPassphrase(keyCheck string, derivedKey []byte) error {
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
	if string(plaintext) != KeyValidationString {
		fmt.Printf("DEBUG: Got unexpected validation content: %s\n", string(plaintext))
		return fmt.Errorf("key validation failed: unexpected content")
	}

	fmt.Printf("DEBUG: Passphrase verification successful\n")
	return nil
}

func verifyPassphraseWithFallback(keyCheck string, derivedKey []byte) error {
	fmt.Printf("keeys %v", derivedKey)
	err := verifyPassphrase(keyCheck, derivedKey)
	if err != nil && strings.Contains(err.Error(), "key check too short") {
		// Try with legacy format (16-byte nonce)
		fmt.Println("Warning: Attempting fallback with 16-byte nonce (legacy format)")
		return verifyLegacyPassphrase(keyCheck, derivedKey)
	}
	return err
}

func verifyLegacyPassphrase(keyCheck string, derivedKey []byte) error {
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
	nonceSize := LegacyNonceSize

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
	if string(plaintext) != KeyValidationString {
		return fmt.Errorf("key validation failed: unexpected content")
	}

	fmt.Println("Warning: Successfully verified with legacy format. Consider reinitializing your vault.")
	return nil
}

// Improved diagnostic function with key/nonce correlation check
func printKeyDetails(cfg *config.VaultConfig) {
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

// Utility functions
func generateNonce() (string, error) {
	nonce := make([]byte, GCMNonceSize) // 12 bytes for GCM
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}

func generateIV() (string, error) {
	iv := make([]byte, CBCIVSize) // 16 bytes for CBC
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(iv), nil
}

func generateSalt() (string, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

func generateRandomKey() ([]byte, error) {
	key := make([]byte, AESKeySize) // 32 bytes for AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func backupKeyToFile(key []byte, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, SecureDirPerms); err != nil {
		return err
	}
	// Write file with restrictive permissions
	return os.WriteFile(path, key, SecureFilePerms)
}

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

func calculateKeyHash(key []byte) string {
	hash := sha256.Sum256(key)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// generateKeyCheck creates a validation string to verify the key during decryption
func generateKeyCheck(key []byte) (string, error) {
	// Create a simple string that can be encrypted and later verified
	plaintext := []byte(KeyValidationString)

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
