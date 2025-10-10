package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/encryption/aesencryption/aeskey"
	"github.com/substantialcattle5/sietch/internal/encryption/chachaencryption/chachakey"
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
	var userPassphrase string
	var err error

	if params.UsePassphrase {
		userPassphrase, err = ui.GetPassphraseForInitialization(cmd, true)
		if err != nil {
			return nil, err
		}
	}

	switch params.KeyType {
	case constants.EncryptionTypeAES:
		return generateAESKey(keyPath, params, userPassphrase)
	case constants.EncryptionTypeChaCha20:
		return generateChaCha20Key(keyPath, params, userPassphrase)
	case constants.EncryptionTypeGPG:
		return generateGPGKey(params, userPassphrase)
	case constants.EncryptionTypeNone:
		// No key generation needed for unencrypted vaults
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported encryption type: %s", params.KeyType)
	}
}

func generateAESKey(keyPath string, params KeyGenParams, userPassphrase string) (*config.KeyConfig, error) {
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

	// Generate the key configuration
	keyConfig, err := aeskey.GenerateAESKey(encConfig, userPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// If we need to save the key to a file
	if encConfig.Encryption.KeyBackupPath != "" {
		fmt.Printf("Key backed up to: %s\n", encConfig.Encryption.KeyBackupPath)
	}

	return keyConfig, nil
}

func generateChaCha20Key(keyPath string, params KeyGenParams, userPassphrase string) (*config.KeyConfig, error) {
	kdfValue := constants.KDFScrypt
	if !params.UseScrypt {
		return nil, fmt.Errorf("ChaCha20 currently only supports scrypt KDF")
	}

	// Create encryption config
	encConfig := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			PassphraseProtected: params.UsePassphrase,
			KeyFile:             params.KeyFile != "",
			KeyFilePath:         params.KeyFile,
			KeyPath:             keyPath,
			ChaChaConfig: &config.ChaChaConfig{
				Mode:    "poly1305",
				KDF:     kdfValue,
				ScryptN: params.ScryptN,
				ScryptR: params.ScryptR,
				ScryptP: params.ScryptP,
			},
		},
	}

	// Generate the key configuration
	keyConfig, err := chachakey.GenerateChaCha20Key(encConfig, userPassphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20 key: %w", err)
	}

	return keyConfig, nil
}

func generateGPGKey(params KeyGenParams, userPassphrase string) (*config.KeyConfig, error) {
	// Check if GPG is available
	if !encryption.IsGPGAvailable() {
		return nil, fmt.Errorf("GPG is not available on this system. Please install GPG first")
	}

	// For non-interactive mode, we need to provide a default GPG key
	// In interactive mode, this would be handled by the prompts
	keys, err := encryption.ListAvailableGPGKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no GPG keys found. Please create a GPG key first using 'gpg --generate-key' or use interactive mode")
	}

	// Use the first available key for non-interactive mode
	// Note: In interactive mode, this key config will be overridden with the user's selection
	selectedKey := keys[0]
	fmt.Printf("Using GPG key: %s (%s)\n", selectedKey.UserID, selectedKey.KeyID)

	// Create vault config for GPG key generation
	vaultConfig := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeGPG,
			PassphraseProtected: params.UsePassphrase,
		},
	}

	// Generate GPG key configuration
	keyConfig, err := encryption.GenerateGPGKeyConfig(vaultConfig, selectedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key configuration: %w", err)
	}

	return keyConfig, nil
}
