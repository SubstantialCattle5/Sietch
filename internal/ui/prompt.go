package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
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
	if err := promptBasicConfig(configuration); err != nil {
		return nil, err
	}

	// Group 2: Security Configuration
	fmt.Println("\n🔹 Security Configuration")
	if err := encryption.PromptSecurityConfig(configuration); err != nil {
		return nil, err
	}

	// Group 3: Chunking & Compression
	fmt.Println("\n🔹 Storage Configuration")
	if err := promptStorageConfig(configuration); err != nil {
		return nil, err
	}

	// Group 4: Metadata
	fmt.Println("\n🔹 Metadata")
	if err := promptMetadataConfig(configuration); err != nil {
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
			return nil, errors.New("operation cancelled")
		}
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return configuration, nil
}

// promptBasicConfig asks for basic vault configuration
func promptBasicConfig(configuration *config.VaultConfig) error {
	namePrompt := promptui.Prompt{
		Label:     "Vault name",
		Default:   "my-sietch",
		AllowEdit: true,
		Validate: func(input string) error {
			if len(input) < 3 {
				return errors.New("vault name must be at least 3 characters")
			}
			return nil
		},
	}

	result, err := namePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Name = result
	return nil
}

// promptStorageConfig asks for chunking, hashing, and compression settings
func promptStorageConfig(configuration *config.VaultConfig) error {
	// Chunking strategy prompt with descriptions
	chunkStrategyPrompt := promptui.Select{
		Label: "Chunking strategy",
		Items: []string{"fixed", "cdc"},
		Templates: &promptui.SelectTemplates{
			Selected: "Chunking strategy: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "fixed" }}Fixed-size chunks (simple and predictable)
{{ else if eq . "cdc" }}Content-Defined Chunking (better deduplication for similar files){{ end }}
`,
		},
	}

	_, chunkResult, err := chunkStrategyPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.Strategy = chunkResult

	// Chunk size prompt with validation
	sizePrompt := promptui.Prompt{
		Label:   "Average chunk size (MB)",
		Default: "4",
		Validate: func(input string) error {
			if len(input) < 1 {
				return errors.New("size must not be empty")
			}
			return nil
		},
	}

	sizeResult, err := sizePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.ChunkSize = sizeResult

	// Hash algorithm prompt with descriptions
	hashAlgorithmPrompt := promptui.Select{
		Label: "Hash algorithm",
		Items: []string{"sha256", "blake3", "sha512", "sha1"},
		Templates: &promptui.SelectTemplates{
			Selected: "Hash algorithm: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "sha256" }}SHA-256 (recommended default, good balance of security and speed)
{{ else if eq . "blake3" }}BLAKE3 (modern, very fast with strong security)
{{ else if eq . "sha512" }}SHA-512 (stronger security, slightly slower)
{{ else if eq . "sha1" }}SHA-1 (faster but less secure, not recommended for sensitive data){{ end }}
`,
		},
	}

	_, hashResult, err := hashAlgorithmPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.HashAlgorithm = hashResult

	// Compression prompt with descriptions
	compressionPrompt := promptui.Select{
		Label: "Compression algorithm",
		Items: []string{"none", "gzip", "zstd"},
		Templates: &promptui.SelectTemplates{
			Selected: "Compression: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "none" }}No compression (faster but larger files)
{{ else if eq . "gzip" }}Gzip compression (good balance of speed/compression)
{{ else if eq . "zstd" }}Zstandard compression (better compression but slower){{ end }}
`,
		},
	}

	_, compResult, err := compressionPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Compression = compResult

	return nil
}

// promptMetadataConfig asks for metadata information
func promptMetadataConfig(configuration *config.VaultConfig) error {
	// Author prompt
	authorPrompt := promptui.Prompt{
		Label:     "Author",
		Default:   "nilay@dune.net",
		AllowEdit: true,
	}

	authorResult, err := authorPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Metadata.Author = authorResult

	// Tags prompt - allow multiple tags
	tagsPrompt := promptui.Prompt{
		Label:     "Tags (comma-separated)",
		Default:   "research,desert,offline",
		AllowEdit: true,
	}

	tagsResult, err := tagsPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Parse comma-separated tags and trim whitespace
	tags := strings.Split(tagsResult, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}
	configuration.Metadata.Tags = tags

	return nil
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
