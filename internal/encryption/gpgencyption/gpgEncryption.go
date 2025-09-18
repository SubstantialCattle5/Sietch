package gpgencyption

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
)

// GPGEncryption encrypts data using GPG with the configured recipient
func GPGEncryption(data string, vaultConfig config.VaultConfig) (string, error) {
	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != "gpg" {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Check if GPG config exists
	if vaultConfig.Encryption.GPGConfig == nil {
		return "", fmt.Errorf("GPG configuration is missing")
	}

	recipient := vaultConfig.Encryption.GPGConfig.Recipient
	if recipient == "" {
		// Try using KeyID as recipient if email is not available
		recipient = vaultConfig.Encryption.GPGConfig.KeyID
	}

	if recipient == "" {
		return "", fmt.Errorf("no recipient configured for GPG encryption")
	}

	// Encrypt using GPG
	encryptedData, err := encryptWithGPG(data, recipient)
	if err != nil {
		return "", fmt.Errorf("GPG encryption failed: %w", err)
	}

	return encryptedData, nil
}

// GPGEncryptionWithPassphrase encrypts data using GPG with passphrase support
func GPGEncryptionWithPassphrase(data string, vaultConfig config.VaultConfig, passphrase string) (string, error) {
	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != "gpg" {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Check if GPG config exists
	if vaultConfig.Encryption.GPGConfig == nil {
		return "", fmt.Errorf("GPG configuration is missing")
	}

	recipient := vaultConfig.Encryption.GPGConfig.Recipient
	if recipient == "" {
		recipient = vaultConfig.Encryption.GPGConfig.KeyID
	}

	if recipient == "" {
		return "", fmt.Errorf("no recipient configured for GPG encryption")
	}

	// Encrypt using GPG with passphrase
	encryptedData, err := encryptWithGPGPassphrase(data, recipient, passphrase)
	if err != nil {
		return "", fmt.Errorf("GPG encryption failed: %w", err)
	}

	return encryptedData, nil
}

// GPGDecryption decrypts GPG-encrypted data
func GPGDecryption(encryptedData string, vaultPath string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != "gpg" {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Decrypt using GPG
	decryptedData, err := decryptWithGPG(encryptedData)
	if err != nil {
		return "", fmt.Errorf("GPG decryption failed: %w", err)
	}

	return decryptedData, nil
}

// GPGDecryptionWithPassphrase decrypts GPG-encrypted data using a passphrase
func GPGDecryptionWithPassphrase(encryptedData string, vaultPath string, passphrase string) (string, error) {
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to load vault config: %w", err)
	}

	// Validate encryption type is GPG
	if vaultConfig.Encryption.Type != "gpg" {
		return "", fmt.Errorf("vault is not configured for GPG encryption (using %s)",
			vaultConfig.Encryption.Type)
	}

	// Decrypt using GPG with passphrase
	decryptedData, err := decryptWithGPGPassphrase(encryptedData, passphrase)
	if err != nil {
		return "", fmt.Errorf("GPG decryption failed: %w", err)
	}

	return decryptedData, nil
}

// encryptWithGPG encrypts data using GPG for the specified recipient
func encryptWithGPG(data, recipient string) (string, error) {
	// Prepare GPG encryption command
	cmd := exec.Command("gpg", "--trust-model", "always", "--armor", "--encrypt", "--recipient", recipient)
	cmd.Stdin = strings.NewReader(data)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute encryption
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("GPG encryption command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// encryptWithGPGPassphrase encrypts data using GPG with passphrase
func encryptWithGPGPassphrase(data, recipient, passphrase string) (string, error) {
	// Prepare GPG encryption command with passphrase
	cmd := exec.Command("gpg", "--trust-model", "always", "--armor", "--encrypt", "--recipient", recipient, "--batch", "--yes", "--passphrase", passphrase)
	cmd.Stdin = strings.NewReader(data)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute encryption
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("GPG encryption command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// decryptWithGPG decrypts GPG-encrypted data
func decryptWithGPG(encryptedData string) (string, error) {
	// Prepare GPG decryption command
	cmd := exec.Command("gpg", "--quiet", "--batch", "--decrypt")
	cmd.Stdin = strings.NewReader(encryptedData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute decryption
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("GPG decryption command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// decryptWithGPGPassphrase decrypts GPG-encrypted data using a passphrase
func decryptWithGPGPassphrase(encryptedData, passphrase string) (string, error) {
	// Prepare GPG decryption command with passphrase
	cmd := exec.Command("gpg", "--quiet", "--batch", "--decrypt", "--passphrase", passphrase)
	cmd.Stdin = strings.NewReader(encryptedData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute decryption
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("GPG decryption command failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ValidateGPGKey validates that a GPG key exists and can be used for encryption
func ValidateGPGKey(keyID string) error {
	// Check if the key exists in the keyring
	cmd := exec.Command("gpg", "--list-keys", keyID)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("GPG key %s not found in keyring: %w", keyID, err)
	}

	return nil
}

// GetGPGKeyFingerprint retrieves the fingerprint for a given key ID
func GetGPGKeyFingerprint(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--with-colons", "--list-keys", keyID)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get key fingerprint: %w", err)
	}

	// Parse output to find fingerprint
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "fpr:") {
			fields := strings.Split(line, ":")
			if len(fields) >= 10 {
				return fields[9], nil
			}
		}
	}

	return "", fmt.Errorf("fingerprint not found for key %s", keyID)
}

// IsGPGAvailable checks if GPG is installed and available
func IsGPGAvailable() bool {
	cmd := exec.Command("gpg", "--version")
	return cmd.Run() == nil
}

// GPGKeyInfo represents information about a GPG key
type GPGKeyInfo struct {
	KeyID       string
	Fingerprint string
	UserID      string
	Email       string
	KeyType     string
	Expired     bool
}

// ListGPGKeys retrieves available GPG keys from the keyring
func ListGPGKeys() ([]*GPGKeyInfo, error) {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gpg --list-keys: %w", err)
	}

	return parseGPGKeyList(string(output)), nil
}

// parseGPGKeyList parses GPG key list output and extracts key information
func parseGPGKeyList(output string) []*GPGKeyInfo {
	var keys []*GPGKeyInfo
	lines := strings.Split(output, "\n")

	var currentKey *GPGKeyInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub":
			// Public key record
			if len(fields) >= 5 {
				currentKey = &GPGKeyInfo{
					KeyID:   fields[4],
					KeyType: fields[3],
					Expired: fields[1] == "e",
				}
			}
		case "fpr":
			// Fingerprint record
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}
		case "uid":
			// User ID record
			if currentKey != nil && len(fields) >= 10 {
				userID := fields[9]
				currentKey.UserID = userID

				// Extract email from user ID using regex
				emailRegex := regexp.MustCompile(`<([^>]+)>`)
				matches := emailRegex.FindStringSubmatch(userID)
				if len(matches) > 1 {
					currentKey.Email = matches[1]
				}

				// Only add the key when we have complete information
				if currentKey.KeyID != "" && !currentKey.Expired {
					keys = append(keys, currentKey)
				}
				currentKey = nil
			}
		}
	}

	return keys
}
