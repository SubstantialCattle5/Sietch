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

	"github.com/substantialcattle5/sietch/internal/config"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
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

	// Generate nonce/IV based on the selected encryption mode
	if cfg.Encryption.AESConfig.Mode == "gcm" || cfg.Encryption.AESConfig.Mode == "" {
		nonce, err := generateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}
		keyConfig.AESConfig.Nonce = nonce
		// Set default mode to GCM if not specified
		if cfg.Encryption.AESConfig.Mode == "" {
			cfg.Encryption.AESConfig.Mode = "gcm"
		}
	} else if cfg.Encryption.AESConfig.Mode == "cbc" {
		iv, err := generateIV()
		if err != nil {
			return nil, fmt.Errorf("failed to generate IV: %w", err)
		}
		keyConfig.AESConfig.IV = iv
	} else {
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
			cfg.Encryption.AESConfig.KDF = "scrypt"
		}
		keyConfig.AESConfig.KDF = cfg.Encryption.AESConfig.KDF

		var derivedKey []byte

		// Generate key using selected KDF
		if cfg.Encryption.AESConfig.KDF == "scrypt" {
			// Set default scrypt parameters if not specified
			if cfg.Encryption.AESConfig.ScryptN == 0 {
				cfg.Encryption.AESConfig.ScryptN = 32768
			}
			if cfg.Encryption.AESConfig.ScryptR == 0 {
				cfg.Encryption.AESConfig.ScryptR = 8
			}
			if cfg.Encryption.AESConfig.ScryptP == 0 {
				cfg.Encryption.AESConfig.ScryptP = 1
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
				32, // 32 bytes for AES-256
			)
			if err != nil {
				return nil, fmt.Errorf("failed to derive key using scrypt: %w", err)
			}
		} else if cfg.Encryption.AESConfig.KDF == "pbkdf2" {
			// Set default PBKDF2 iterations if not specified
			if cfg.Encryption.AESConfig.PBKDF2I == 0 {
				cfg.Encryption.AESConfig.PBKDF2I = 10000
			}

			keyConfig.AESConfig.PBKDF2I = cfg.Encryption.AESConfig.PBKDF2I

			// Generate key using PBKDF2
			derivedKey = pbkdf2.Key(
				[]byte(passphrase),
				[]byte(salt),
				cfg.Encryption.AESConfig.PBKDF2I,
				32, // 32 bytes for AES-256
				sha256.New,
			)
		} else {
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
			if keyLen != 16 && keyLen != 24 && keyLen != 32 {
				return nil, fmt.Errorf("invalid key length %d bytes - must be 16, 24, or 32 bytes for AES", keyLen)
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
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
		}

		// Write the key with secure permissions
		if err := os.WriteFile(cfg.Encryption.KeyPath, keyMaterial, 0600); err != nil {
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

		// Encrypt the key material
		encryptedKey := gcm.Seal(nil, nonce, keyMaterial, nil)

		// Prepend nonce for storage
		return append(nonce, encryptedKey...), nil

	case "cbc":
		// For CBC mode
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
		mode := cipher.NewCBCEncrypter(block, iv)
		ciphertext := make([]byte, len(paddedData))
		mode.CryptBlocks(ciphertext, paddedData)

		// Prepend IV for storage
		return append(iv, ciphertext...), nil

	default:
		return nil, fmt.Errorf("unsupported encryption mode: '%s' (must be 'gcm' or 'cbc')", mode)
	}
}

func generateNonce() (string, error) {
	nonce := make([]byte, 12) // 12 bytes for GCM
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonce), nil
}

func generateIV() (string, error) {
	iv := make([]byte, 16) // 16 bytes for CBC
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(iv), nil
}

func generateSalt() (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

func generateRandomKey() ([]byte, error) {
	key := make([]byte, 32) // 32 bytes for AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func backupKeyToFile(key []byte, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// Write file with restrictive permissions
	return os.WriteFile(path, key, 0600)
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
	plaintext := []byte("sietch-key-validation")

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
