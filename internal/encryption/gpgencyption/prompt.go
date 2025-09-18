package gpgencyption

import (
	"fmt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/gpgencyption/gpgkey"
	"github.com/substantialcattle5/sietch/internal/encryption/passphrase"
)

//*
//* really good article on gpg saving for reference,
//* https://www.gnupg.org/documentation/manuals/gnupg/GPG-Configuration.html
//*

func PromptGPGOptions(configuration *config.VaultConfig) error {
	// Initialize GPG config if not exists
	if configuration.Encryption.GPGConfig == nil {
		configuration.Encryption.GPGConfig = &config.GPGConfig{}
	}

	// Check if GPG is available
	if !gpgkey.IsGPGAvailable() {
		return fmt.Errorf("GPG is not available on this system. Please install GPG first")
	}

	// Get available GPG keys
	keys, err := gpgkey.ListGPGKeys()
	if err != nil {
		return fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Prompt for key selection or creation
	selectedKey, err := gpgkey.PromptForKeySelection(keys)
	if err != nil {
		return err
	}

	if selectedKey == nil {
		// User chose to create a new key
		newKey, err := gpgkey.PromptForNewKeyCreation()
		if err != nil {
			return err
		}
		selectedKey = newKey
	}

	// Store the selected key configuration
	configuration.Encryption.GPGConfig.KeyID = selectedKey.KeyID
	configuration.Encryption.GPGConfig.Recipient = selectedKey.Email
	configuration.Encryption.KeyHash = selectedKey.Fingerprint

	// Prompt for passphrase protection (for private key access)
	if err := passphrase.PromptPassphraseProtection(configuration); err != nil {
		return err
	}

	// Configure key server (optional)
	if err := gpgkey.PromptForKeyServer(configuration); err != nil {
		return err
	}

	fmt.Printf("✓ GPG key configured: %s (%s)\n", selectedKey.UserID, selectedKey.KeyID)

	return nil
}
