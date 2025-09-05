package aeskey

import (
	"fmt"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
)

// promptKeyFileOptions handles configuration when not using passphrase protection
func PromptKeyFileOptions(configuration *config.VaultConfig) error {
	// Ask if user wants to use an existing key file
	keyFilePrompt := promptui.Prompt{
		Label:     "Use existing key file",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := keyFilePrompt.Run()
	if err == nil { // User selected yes
		return promptExistingKeyFile(configuration)
	}
	return promptRandomKeyGeneration(configuration)
}

// promptExistingKeyFile handles configuration for using an existing key file
func promptExistingKeyFile(configuration *config.VaultConfig) error {
	configuration.Encryption.KeyFile = true

	// Get path to key file
	pathPrompt := promptui.Prompt{
		Label:     "Path to key file",
		Default:   "~/.keys/sietch.key",
		AllowEdit: true,
	}

	keyPath, err := pathPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.KeyFilePath = keyPath

	return nil
}

// promptRandomKeyGeneration handles configuration for random key generation
func promptRandomKeyGeneration(configuration *config.VaultConfig) error {
	// Random key generation
	configuration.Encryption.RandomKey = true

	// Ask about backing up the key
	backupPrompt := promptui.Prompt{
		Label:     "Backup random key to file",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := backupPrompt.Run()
	if err == nil { // User selected yes
		return promptKeyBackupPath(configuration)
	}

	// Make sure KeyBackupPath is empty if not backing up
	configuration.Encryption.KeyBackupPath = ""
	return nil
}

// promptKeyBackupPath handles configuration for key backup path
func promptKeyBackupPath(configuration *config.VaultConfig) error {
	pathPrompt := promptui.Prompt{
		Label:     "Backup file path",
		Default:   "~/sietch-key-backup.bin",
		AllowEdit: true,
	}

	backupPath, err := pathPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.KeyBackupPath = backupPath
	return nil
}

// promptKDFOptions handles configuration of the key derivation function
func PromptKDFOptions(configuration *config.VaultConfig) error {
	kdfPrompt := promptui.Select{
		Label: "Key derivation function",
		Items: []string{"scrypt", "pbkdf2"},
		Templates: &promptui.SelectTemplates{
			Selected: "KDF: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "scrypt" }}Scrypt (memory-hard, recommended)
{{ else if eq . "pbkdf2" }}PBKDF2 (more compatible, less secure){{ end }}
`,
		},
	}

	_, kdf, err := kdfPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.AESConfig.KDF = kdf

	if kdf == "scrypt" {
		return promptScryptParameters(configuration)
	}
	return promptPBKDF2Parameters(configuration)
}

// promptScryptParameters handles configuration of scrypt parameters
func promptScryptParameters(configuration *config.VaultConfig) error {
	advancedPrompt := promptui.Prompt{
		Label:     "Configure advanced scrypt parameters",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := advancedPrompt.Run()
	if err == nil { // User selected yes
		return promptAdvancedScryptParameters(configuration)
	}
	// Default scrypt parameters
	configuration.Encryption.AESConfig.ScryptN = 32768
	configuration.Encryption.AESConfig.ScryptR = 8
	configuration.Encryption.AESConfig.ScryptP = 1
	return nil
}

// promptAdvancedScryptParameters handles configuration of advanced scrypt parameters
func promptAdvancedScryptParameters(configuration *config.VaultConfig) error {
	// Scrypt N parameter
	nPrompt := promptui.Select{
		Label: "Scrypt N parameter (CPU/memory cost)",
		Items: []string{"16384", "32768", "65536", "131072"},
		Templates: &promptui.SelectTemplates{
			Selected: "N: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 16384: Fast, lower security
- 32768: Balanced (recommended)
- 65536: More secure, slower
- 131072: Most secure, much slower
`,
		},
	}

	nIdx, _, err := nPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	nValues := []int{16384, 32768, 65536, 131072}
	configuration.Encryption.AESConfig.ScryptN = nValues[nIdx]

	// Scrypt r and p parameters
	configuration.Encryption.AESConfig.ScryptR = 8
	configuration.Encryption.AESConfig.ScryptP = 1

	return nil
}

// promptPBKDF2Parameters handles configuration of PBKDF2 parameters
func promptPBKDF2Parameters(configuration *config.VaultConfig) error {
	// PBKDF2 iterations
	iterPrompt := promptui.Select{
		Label: "PBKDF2 iterations",
		Items: []string{"100000", "200000", "500000", "1000000"},
		Templates: &promptui.SelectTemplates{
			Selected: "Iterations: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 100000: Fast, lower security
- 200000: Balanced (recommended)
- 500000: More secure, slower
- 1000000: Most secure, much slower
`,
		},
	}

	iterIdx, _, err := iterPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	iterValues := []int{100000, 200000, 500000, 1000000}
	configuration.Encryption.AESConfig.PBKDF2I = iterValues[iterIdx]

	return nil
}
