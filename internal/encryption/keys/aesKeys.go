package keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// GenerateAESKey creates a key configuration based on vault settings
func GenerateAESKey(cfg *config.VaultConfig, passphrase string) (*config.KeyConfig, error) {
	keyConfig := &config.KeyConfig{
		AESConfig: &config.AESConfig{},
	}

	// Generate nonce/IV based on the selected encryption mode
	if cfg.Encryption.AESConfig.Mode == "gcm" {
		nonce, err := generateNonce()
		if err != nil {
			return nil, fmt.Errorf("failed to generate nonce: %w", err)
		}
		keyConfig.AESConfig.Nonce = nonce
	} else {
		// CBC mode
		iv, err := generateIV()
		if err != nil {
			return nil, fmt.Errorf("failed to generate IV: %w", err)
		}
		keyConfig.AESConfig.IV = iv
	}

	// Handle passphrase-protected encryption
	if cfg.Encryption.PassphraseProtected {
		salt, err := generateSalt()
		if err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
		keyConfig.Salt = salt

		// Copy KDF algorithm type from config
		keyConfig.AESConfig.KDF = cfg.Encryption.AESConfig.KDF

		// Copy KDF parameters from config
		if cfg.Encryption.AESConfig.KDF == "scrypt" {
			keyConfig.AESConfig.ScryptN = cfg.Encryption.AESConfig.ScryptN
			keyConfig.AESConfig.ScryptR = cfg.Encryption.AESConfig.ScryptR
			keyConfig.AESConfig.ScryptP = cfg.Encryption.AESConfig.ScryptP

			// Generate key using scrypt
			key, err := scrypt.Key(
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
			keyConfig.KeyHash = calculateKeyHash(key)
		} else {
			// PBKDF2
			keyConfig.AESConfig.PBKDF2I = cfg.Encryption.AESConfig.PBKDF2I

			// Generate key using PBKDF2
			key := pbkdf2.Key(
				[]byte(passphrase),
				[]byte(salt),
				cfg.Encryption.AESConfig.PBKDF2I,
				32, // 32 bytes for AES-256
				sha256.New,
			)
			keyConfig.KeyHash = calculateKeyHash(key)
		}

		return keyConfig, nil
	}

	// Handle key file or random key generation
	var key []byte
	var err error

	if cfg.Encryption.KeyFile {
		// Use existing key file
		expandedPath := expandPath(cfg.Encryption.KeyFilePath)
		key, err = os.ReadFile(expandedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file %s: %w", expandedPath, err)
		}
	} else {
		// Generate random key
		key, err = generateRandomKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate random key: %w", err)
		}

		// Backup key if requested
		if cfg.Encryption.KeyBackupPath != "" {
			expandedBackupPath := expandPath(cfg.Encryption.KeyBackupPath)
			if err := backupKeyToFile(key, expandedBackupPath); err != nil {
				return nil, fmt.Errorf("failed to backup key to %s: %w", expandedBackupPath, err)
			}
			fmt.Printf("Key backed up to: %s\n", expandedBackupPath)
		}
	}

	keyConfig.KeyHash = calculateKeyHash(key)
	return keyConfig, nil
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
