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

// displayConfigSummary shows a summary of the configuration
func displayConfigSummary(configuration *config.VaultConfig) {
	fmt.Println("\n📋 Configuration Summary")
	fmt.Println("========================")
	fmt.Printf("Vault Name: %s\n", configuration.Name)
	fmt.Printf("Encryption: %s", configuration.Encryption.Type)
	if configuration.Encryption.PassphraseProtected {
		fmt.Printf(" (passphrase protected)")
	}
	fmt.Println()
	fmt.Printf("Chunking: %s (avg. %s MB)\n", configuration.Chunking.Strategy, configuration.Chunking.ChunkSize)
	fmt.Printf("Compression: %s\n", configuration.Chunking.HashAlgorithm)
	fmt.Printf("Author: %s\n", configuration.Metadata.Author)
	fmt.Printf("Tags: %s\n", strings.Join(configuration.Metadata.Tags, ", "))
	fmt.Println()
}
