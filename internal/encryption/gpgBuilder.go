package encryption

import (
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/encryption/gpgencyption"
)

// GPGEncryption encrypts data using GPG with the configured recipient
func GPGEncryption(data string, vaultConfig config.VaultConfig) (string, error) {
	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != constants.EncryptionTypeGPG {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	return gpgencyption.GPGEncryption(data, vaultConfig)
}

// GPGEncryptWithPassphrase encrypts data using GPG with passphrase support
func GPGEncryptWithPassphrase(data string, vaultConfig config.VaultConfig, passphrase string) (string, error) {
	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != constants.EncryptionTypeGPG {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Choose encryption method based on passphrase protection
	if vaultConfig.Encryption.PassphraseProtected {
		return gpgencyption.GPGEncryptionWithPassphrase(data, vaultConfig, passphrase)
	} else {
		return gpgencyption.GPGEncryption(data, vaultConfig)
	}
}

// GPGDecryption decrypts GPG-encrypted data
func GPGDecryption(encryptedData string, vaultPath string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != constants.EncryptionTypeGPG {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	return gpgencyption.GPGDecryption(encryptedData, vaultPath)
}

// GPGDecryptionWithPassphrase decrypts GPG-encrypted data using a passphrase
func GPGDecryptionWithPassphrase(encryptedData string, vaultPath string, passphrase string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != constants.EncryptionTypeGPG {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Choose decryption method based on passphrase protection
	if vaultConfig.Encryption.PassphraseProtected {
		return gpgencyption.GPGDecryptionWithPassphrase(encryptedData, vaultPath, passphrase)
	} else {
		return gpgencyption.GPGDecryption(encryptedData, vaultPath)
	}
}

// ValidateGPGConfiguration validates the GPG configuration
func ValidateGPGConfiguration(vaultConfig config.VaultConfig) error {
	if vaultConfig.Encryption.Type != constants.EncryptionTypeGPG {
		return fmt.Errorf("not a GPG vault")
	}

	if vaultConfig.Encryption.GPGConfig == nil {
		return fmt.Errorf("GPG configuration is missing")
	}

	gpgConfig := vaultConfig.Encryption.GPGConfig

	// Check if we have either KeyID or Recipient
	if gpgConfig.KeyID == "" && gpgConfig.Recipient == "" {
		return fmt.Errorf("either KeyID or Recipient must be specified")
	}

	// Validate key exists if KeyID is provided
	if gpgConfig.KeyID != "" {
		if err := gpgencyption.ValidateGPGKey(gpgConfig.KeyID); err != nil {
			return fmt.Errorf("GPG key validation failed: %w", err)
		}
	}

	return nil
}

// GetGPGKeyInfo retrieves detailed information about the configured GPG key
func GetGPGKeyInfo(vaultConfig config.VaultConfig) (*GPGKeyDetails, error) {
	if err := ValidateGPGConfiguration(vaultConfig); err != nil {
		return nil, err
	}

	keyID := vaultConfig.Encryption.GPGConfig.KeyID
	if keyID == "" {
		keyID = vaultConfig.Encryption.GPGConfig.Recipient
	}

	fingerprint, err := gpgencyption.GetGPGKeyFingerprint(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key fingerprint: %w", err)
	}

	return &GPGKeyDetails{
		KeyID:       keyID,
		Fingerprint: fingerprint,
		Recipient:   vaultConfig.Encryption.GPGConfig.Recipient,
		KeyServer:   vaultConfig.Encryption.GPGConfig.KeyServer,
	}, nil
}

// GPGKeyDetails contains detailed information about a GPG key
type GPGKeyDetails struct {
	KeyID       string
	Fingerprint string
	Recipient   string
	KeyServer   string
}

// String returns a human-readable representation of the GPG key details
func (g *GPGKeyDetails) String() string {
	var details []string

	if g.KeyID != "" {
		details = append(details, fmt.Sprintf("Key ID: %s", g.KeyID))
	}

	if g.Fingerprint != "" {
		details = append(details, fmt.Sprintf("Fingerprint: %s", g.Fingerprint))
	}

	if g.Recipient != "" {
		details = append(details, fmt.Sprintf("Recipient: %s", g.Recipient))
	}

	if g.KeyServer != "" {
		details = append(details, fmt.Sprintf("Key Server: %s", g.KeyServer))
	}

	return strings.Join(details, "\n")
}

// IsGPGAvailable checks if GPG is available on the system
func IsGPGAvailable() bool {
	return gpgencyption.IsGPGAvailable()
}

// ListAvailableGPGKeys returns a list of available GPG keys
func ListAvailableGPGKeys() ([]*gpgencyption.GPGKeyInfo, error) {
	return gpgencyption.ListGPGKeys()
}

// GenerateGPGKeyConfig creates a GPG key configuration for vault initialization
func GenerateGPGKeyConfig(vaultConfig *config.VaultConfig, selectedKey *gpgencyption.GPGKeyInfo) (*config.KeyConfig, error) {
	if selectedKey == nil {
		return nil, fmt.Errorf("no GPG key provided")
	}

	// Validate the key exists and is usable
	if err := gpgencyption.ValidateGPGKey(selectedKey.KeyID); err != nil {
		return nil, fmt.Errorf("GPG key validation failed: %w", err)
	}

	// Get fingerprint if not already available
	fingerprint := selectedKey.Fingerprint
	if fingerprint == "" {
		var err error
		fingerprint, err = gpgencyption.GetGPGKeyFingerprint(selectedKey.KeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get key fingerprint: %w", err)
		}
	}

	// Create GPG configuration
	gpgConfig := &config.GPGConfig{
		KeyID:     selectedKey.KeyID,
		Recipient: selectedKey.Email,
		KeyServer: "hkps://keys.openpgp.org", // Default key server
	}

	// If vault config already has a key server, use it
	if vaultConfig.Encryption.GPGConfig != nil && vaultConfig.Encryption.GPGConfig.KeyServer != "" {
		gpgConfig.KeyServer = vaultConfig.Encryption.GPGConfig.KeyServer
	}

	// Create key configuration
	keyConfig := &config.KeyConfig{
		KeyHash:   fingerprint,
		GPGConfig: gpgConfig,
	}

	return keyConfig, nil
}

// SetupGPGEncryption configures GPG encryption for a vault
func SetupGPGEncryption(vaultConfig *config.VaultConfig, keyConfig *config.KeyConfig) error {
	if keyConfig.GPGConfig == nil {
		return fmt.Errorf("GPG configuration is missing from key config")
	}

	// Set encryption type
	vaultConfig.Encryption.Type = constants.EncryptionTypeGPG

	// Copy GPG configuration
	vaultConfig.Encryption.GPGConfig = keyConfig.GPGConfig
	vaultConfig.Encryption.KeyHash = keyConfig.KeyHash

	// Validate the configuration
	if err := ValidateGPGConfiguration(*vaultConfig); err != nil {
		return fmt.Errorf("GPG configuration validation failed: %w", err)
	}

	return nil
}
