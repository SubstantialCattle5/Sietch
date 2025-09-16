package aeskey

import (
	"fmt"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// PromptKeyFileOptions handles configuration when not using passphrase protection
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

// PromptKDFOptions handles configuration of the key derivation function, provides options for scrypt and pbkdf2
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

	if kdf == constants.KDFScrypt {
		return PromptScryptParameters(configuration)
	}
	return PromptPBKDF2Parameters(configuration)
}
