package encryption

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
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

	switch configuration.Encryption.Type {
	case constants.EncryptionTypeAES:
		if err := aesencryption.PromptAESOptions(configuration); err != nil {
			return err
		}
	case constants.EncryptionTypeGPG:
		if err := gpgkey.PromptGPGOptions(configuration); err != nil {
			return err
		}
	//TODO: Add ChaCha20-Poly1305 encryption
	default:
		fmt.Println("⚠️  Warning: No encryption will be used. Only suitable for testing.")
	}
	return nil
}

// promptEncryptionType asks the user to select an encryption type
func promptEncryptionType(configuration *config.VaultConfig) error {
	keyTypePrompt := promptui.Select{
		Label: "Encryption type",
		Items: []string{"aes", "gpg", "chacha20", "none"},
		Templates: &promptui.SelectTemplates{
			Selected: "Encryption type: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "aes" }}AES-256 encryption (recommended for most users)
{{ else if eq . "gpg" }}GPG encryption (use your existing GPG keys)
 {{ else if eq . "chacha20" }} ChaCha20-Poly1305 encryption (used for systems without dedicated AES hardware acceleration.)
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
