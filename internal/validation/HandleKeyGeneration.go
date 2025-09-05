package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
	"github.com/substantialcattle5/sietch/internal/ui"
)

// KeyGenParams holds all parameters needed for key generation
type KeyGenParams struct {
	KeyType          string
	UsePassphrase    bool
	KeyFile          string
	AESMode          string
	UseScrypt        bool
	ScryptN          int
	ScryptR          int
	ScryptP          int
	PBKDF2Iterations int
}

// HandleKeyGeneration manages key generation or import for a vault
func HandleKeyGeneration(cmd *cobra.Command, absVaultPath string, params KeyGenParams) (*config.KeyConfig, error) {
	keyPath := filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")
	var keyConfig *config.KeyConfig
	var err error

	if params.KeyFile != "" {
		// Import key from file
		if err := importKeyFromFile(params.KeyFile, keyPath); err != nil {
			return nil, err
		}
		fmt.Printf("Imported key from %s\n", params.KeyFile)
	} else {
		// Generate new key
		keyConfig, err = generateNewKey(cmd, keyPath, params)
		if err != nil {
			return nil, err
		}
	}

	return keyConfig, nil
}

func importKeyFromFile(sourceKeyFile, destKeyPath string) error {
	// Read key data from source file
	keyData, err := os.ReadFile(sourceKeyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file %s: %w", sourceKeyFile, err)
	}

	// Ensure directory exists
	keyDir := filepath.Dir(destKeyPath)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write key file with secure permissions
	if err := os.WriteFile(destKeyPath, keyData, 0o600); err != nil {
		return fmt.Errorf("failed to write key to %s: %w", destKeyPath, err)
	}

	return nil
}

func generateNewKey(cmd *cobra.Command, keyPath string, params KeyGenParams) (*config.KeyConfig, error) {
	kdfValue := "pbkdf2"
	if params.UseScrypt {
		kdfValue = "scrypt"
	}

	// Create encryption config
	encConfig := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                params.KeyType,
			PassphraseProtected: params.UsePassphrase,
			KeyFile:             params.KeyFile != "",
			KeyFilePath:         params.KeyFile,
			KeyPath:             keyPath,
			AESConfig: &config.AESConfig{
				Mode:    params.AESMode,
				KDF:     kdfValue,
				ScryptN: params.ScryptN,
				ScryptR: params.ScryptR,
				ScryptP: params.ScryptP,
				PBKDF2I: params.PBKDF2Iterations,
			},
		},
	}

	var userPassphrase string
	var err error

	if params.UsePassphrase {
		userPassphrase, err = ui.GetPassphraseForInitialization(cmd, true)
		if err != nil {
			return nil, err
		}
	}

	// Generate the key configuration
	keyConfig, err := keys.GenerateAESKey(encConfig, userPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// If we need to save the key to a file
	if encConfig.Encryption.KeyBackupPath != "" {
		fmt.Printf("Key backed up to: %s\n", encConfig.Encryption.KeyBackupPath)
	}

	return keyConfig, nil
}
