package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/vault"
)

// PromptForInputs guides the user through setting up a vault configuration
func PromptForInputs() (*config.VaultConfig, error) {
	configuration := &config.VaultConfig{}

	// Display welcome message
	fmt.Println("📦 Setting up your Sietch Vault")
	fmt.Println("===============================")
	fmt.Println("Let's configure your secure vault with the following steps:")
	fmt.Println()

	// Group 1: Basic Configuration
	fmt.Println("🔹 Basic Configuration")
	if err := vault.PromptBasicConfig(configuration); err != nil {
		return nil, err
	}

	// Group 2: Security Configuration
	fmt.Println("\n🔹 Security Configuration")
	if err := encryption.PromptSecurityConfig(configuration); err != nil {
		return nil, err
	}

	// Group 3: Chunking & Compression
	fmt.Println("\n🔹 Storage Configuration")
	if err := chunk.PromptStorageConfig(configuration); err != nil {
		return nil, err
	}

	// Group 4: Metadata
	fmt.Println("\n🔹 Metadata")
	if err := vault.PromptMetadataConfig(configuration); err != nil {
		return nil, err
	}

	// Display summary before confirmation
	displayConfigSummary(configuration)

	// Final confirmation
	confirmPrompt := promptui.Prompt{
		Label:     "Create vault with these settings",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := confirmPrompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return nil, errors.New("operation canceled")
		}
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return configuration, nil
}

// displayConfigSummary shows a clean summary of the configuration
func displayConfigSummary(configuration *config.VaultConfig) {
	fmt.Println()
	fmt.Println("📋 Configuration Summary")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	// Basic Information
	fmt.Println("🏷️  Basic Information:")
	fmt.Printf("   • Vault Name: %s\n", configuration.Name)
	fmt.Printf("   • Author:     %s\n", configuration.Metadata.Author)

	tagsStr := strings.Join(configuration.Metadata.Tags, ", ")
	if tagsStr == "" {
		tagsStr = "none"
	}
	fmt.Printf("   • Tags:       %s\n", tagsStr)
	fmt.Println()

	// Security Configuration
	fmt.Println("🔐 Security:")
	encryptionDesc := getEncryptionDescription(configuration.Encryption)
	fmt.Printf("   • Encryption: %s\n", encryptionDesc)
	fmt.Println()

	// Storage Configuration
	fmt.Println("💾 Storage:")
	fmt.Printf("   • Chunking:    %s\n", configuration.Chunking.Strategy)
	fmt.Printf("   • Chunk Size:  %s\n", configuration.Chunking.ChunkSize)
	fmt.Printf("   • Hash Algo:   %s\n", configuration.Chunking.HashAlgorithm)

	compressionDesc := configuration.Compression
	if compressionDesc == "" {
		compressionDesc = "none"
	}
	fmt.Printf("   • Compression: %s\n", compressionDesc)

	fmt.Println()
	fmt.Println(strings.Repeat("=", 52))
}

// getEncryptionDescription returns a human-readable description of the encryption config
func getEncryptionDescription(enc config.EncryptionConfig) string {
	if enc.Type == "" || enc.Type == "none" {
		return "None ⚠️  (not recommended)"
	}

	desc := strings.ToUpper(enc.Type)
	if enc.PassphraseProtected {
		desc += " 🔒 (passphrase protected)"
	}

	// Add specific details for different encryption types
	switch enc.Type {
	case "aes":
		if enc.AESConfig != nil && enc.AESConfig.Mode != "" {
			desc += "-" + strings.ToUpper(enc.AESConfig.Mode)
		} else {
			desc += "-GCM" // default mode
		}
	case "gpg":
		if enc.GPGConfig != nil && enc.GPGConfig.KeyID != "" {
			keyID := enc.GPGConfig.KeyID
			if len(keyID) > 8 {
				keyID = keyID[:8]
			}
			desc += fmt.Sprintf(" (Key: %s)", keyID)
		}
	}

	return desc
}
