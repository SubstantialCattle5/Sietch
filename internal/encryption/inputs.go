package encryption

import (
	"fmt"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/aesencryption"
	"github.com/substantialcattle5/sietch/internal/encryption/gpgencyption/gpgkey"
)

// PromptSecurityConfig asks for security-related configuration
func PromptSecurityConfig(configuration *config.VaultConfig) error {
	// Initialize encryption config if not exists
	if configuration.Encryption.Type == "" {
		configuration.Encryption = config.EncryptionConfig{}
	}

	if err := promptEncryptionType(configuration); err != nil {
		return err
	}

	if configuration.Encryption.Type == "aes" {
		if err := aesencryption.PromptAESOptions(configuration); err != nil {
			return err
		}
	} else if configuration.Encryption.Type == "gpg" {
		if err := gpgkey.PromptGPGOptions(configuration); err != nil {
			return err
		}
	} else {
		fmt.Println("⚠️  Warning: No encryption will be used. Only suitable for testing.")
	}
	return nil
}

// promptEncryptionType asks the user to select an encryption type
func promptEncryptionType(configuration *config.VaultConfig) error {
	keyTypePrompt := promptui.Select{
		Label: "Encryption type",
		Items: []string{"aes", "gpg", "none"},
		Templates: &promptui.SelectTemplates{
			Selected: "Encryption type: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "aes" }}AES-256 encryption (recommended for most users)
{{ else if eq . "gpg" }}GPG encryption (use your existing GPG keys)
{{ else if eq . "none" }}No encryption (not recommended for sensitive data){{ end }}
`,
		},
	}

	_, keyTypeResult, err := keyTypePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.Type = keyTypeResult
	return nil
}
