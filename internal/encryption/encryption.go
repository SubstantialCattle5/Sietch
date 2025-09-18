package encryption

import (
	"fmt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// EncryptData encrypts data using the configured encryption method
func EncryptData(data string, vaultConfig config.VaultConfig) (string, error) {
	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		return AesEncryption(data, vaultConfig)
	case constants.EncryptionTypeGPG:
		return GPGEncryption(data, vaultConfig)
	case constants.EncryptionTypeNone:
		return data, nil
	default:
		return "", fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}

// EncryptDataWithPassphrase encrypts data using the configured encryption method with passphrase
func EncryptDataWithPassphrase(data string, vaultConfig config.VaultConfig, passphrase string) (string, error) {
	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		return AesEncryptWithPassphrase(data, vaultConfig, passphrase)
	case constants.EncryptionTypeGPG:
		return GPGEncryptWithPassphrase(data, vaultConfig, passphrase)
	case constants.EncryptionTypeNone:
		return data, nil
	default:
		return "", fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}

// DecryptData decrypts data using the configured encryption method
func DecryptData(encryptedData string, vaultPath string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		return AesDecryption(encryptedData, vaultPath)
	case constants.EncryptionTypeGPG:
		return GPGDecryption(encryptedData, vaultPath)
	case constants.EncryptionTypeNone:
		return encryptedData, nil
	default:
		return "", fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}

// DecryptDataWithPassphrase decrypts data using the configured encryption method with passphrase
func DecryptDataWithPassphrase(encryptedData string, vaultPath string, passphrase string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		return AesDecryptionWithPassphrase(encryptedData, vaultPath, passphrase)
	case constants.EncryptionTypeGPG:
		return GPGDecryptionWithPassphrase(encryptedData, vaultPath, passphrase)
	case constants.EncryptionTypeNone:
		return encryptedData, nil
	default:
		return "", fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}

// ValidateEncryptionConfiguration validates the encryption configuration
func ValidateEncryptionConfiguration(vaultConfig config.VaultConfig) error {
	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		// AES validation logic would go here
		if vaultConfig.Encryption.AESConfig == nil {
			return fmt.Errorf("AES configuration is missing")
		}
		return nil
	case constants.EncryptionTypeGPG:
		return ValidateGPGConfiguration(vaultConfig)
	case constants.EncryptionTypeNone:
		return nil
	default:
		return fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}

// GetEncryptionDetails returns human-readable encryption details
func GetEncryptionDetails(vaultConfig config.VaultConfig) (string, error) {
	switch vaultConfig.Encryption.Type {
	case constants.EncryptionTypeAES:
		if vaultConfig.Encryption.AESConfig == nil {
			return "AES (configuration missing)", nil
		}
		mode := vaultConfig.Encryption.AESConfig.Mode
		if mode == "" {
			mode = "GCM"
		}
		return fmt.Sprintf("AES-%s", mode), nil
	case constants.EncryptionTypeGPG:
		gpgDetails, err := GetGPGKeyInfo(vaultConfig)
		if err != nil {
			return "GPG (configuration error)", nil
		}
		return fmt.Sprintf("GPG (Key: %s)", gpgDetails.KeyID), nil
	case constants.EncryptionTypeNone:
		return "None (unencrypted)", nil
	default:
		return "Unknown", fmt.Errorf("unsupported encryption type: %s", vaultConfig.Encryption.Type)
	}
}
