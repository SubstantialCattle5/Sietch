package chunk

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
)

// PromptStorageConfig asks for chunking, hashing, and compression settings
func PromptStorageConfig(configuration *config.VaultConfig) error {
	if err := PromptChunkingConfig(configuration); err != nil {
		return err
	}

	if err := PromptCompressionConfig(configuration); err != nil {
		return err
	}

	return nil
}

// PromptChunkingConfig asks for chunking and hashing settings
func PromptChunkingConfig(configuration *config.VaultConfig) error {
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
		Label:   "Average chunk size",
		Default: "4MB",
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

	return nil
}

// PromptCompressionConfig asks for compression settings
func PromptCompressionConfig(configuration *config.VaultConfig) error {
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
