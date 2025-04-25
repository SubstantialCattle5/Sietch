package aesencryption

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/aesencryption/aeskey"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
	"github.com/substantialcattle5/sietch/internal/encryption/passphrase"
)

func PromptAESOptions(configuration *config.VaultConfig) error {
	// Initialize AES config if not exists
	if configuration.Encryption.AESConfig == nil {
		configuration.Encryption.AESConfig = &config.AESConfig{}
	}

	if err := promptAESMode(configuration); err != nil {
		return err
	}

	if err := passphrase.PromptPassphraseProtection(configuration); err != nil {
		return err
	}

	if configuration.Encryption.PassphraseProtected {
		if err := aeskey.PromptKDFOptions(configuration); err != nil {
			return err
		}

		// Prompt for passphrase
		passphrase, err := passphrase.PromptForPassphrase(true)
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %w", err)

		}
		// Generate key configuration
		keyConfig, err := keys.GenerateAESKey(configuration, passphrase)
		if err != nil {
			return fmt.Errorf("failed to generate AES key: %w", err)
		}

		// Store key configuration
		configuration.Encryption.AESConfig = keyConfig.AESConfig
		configuration.Encryption.KeyHash = keyConfig.KeyHash
		configuration.Encryption.AESConfig.Salt = keyConfig.Salt
	} else {
		if err := aeskey.PromptKeyFileOptions(configuration); err != nil {
			return err
		}

		// Generate key configuration for key file or random key
		keyConfig, err := keys.GenerateAESKey(configuration, "")
		if err != nil {
			return fmt.Errorf("failed to generate AES key: %w", err)
		}

		// Store key configuration
		configuration.Encryption.AESConfig = keyConfig.AESConfig
		configuration.Encryption.KeyHash = keyConfig.KeyHash
	}

	return nil
}

// promptAESMode asks for AES encryption mode
func promptAESMode(configuration *config.VaultConfig) error {
	aesModePrompt := promptui.Select{
		Label: "AES encryption mode",
		Items: []string{"gcm", "cbc"},
		Templates: &promptui.SelectTemplates{
			Selected: "AES mode: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "gcm" }}GCM mode (authenticated encryption, recommended)
{{ else if eq . "cbc" }}CBC mode (compatibility with older systems){{ end }}
`,
		},
	}

	_, aesMode, err := aesModePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.AESConfig.Mode = aesMode
	return nil
}
