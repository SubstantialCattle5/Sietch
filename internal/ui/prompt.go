package ui

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
)

func PromptForInputs() (*config.VaultConfig, error) {
	configuration := &config.VaultConfig{}

	namePrompt := promptui.Prompt{
		Label:     "Vault name",
		Default:   "my-sietch",
		AllowEdit: true,
	}

	result, err := namePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Name = result

	// Key type prompt
	keyTypePrompt := promptui.Select{
		Label: "Key type",
		Items: []string{"aes", "gpg", "none"},
		Templates: &promptui.SelectTemplates{
			Selected: "Key type: {{ . }}",
		},
	}

	_, keyTypeResult, err := keyTypePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.Type = keyTypeResult

	// // Passphrase protection prompt (only if key type isn't "none")
	// if configuration.Encryption.Type != "none" {
	// 	passphrasePrompt := promptui.Prompt{
	// 		Label:     "Protect AES key with passphrase",
	// 		IsConfirm: true,
	// 		Default:   "n",
	// 	}

	// 	_, err := passphrasePrompt.Run()
	// 	if err == nil {
	// 		configuration.Encryption.PassphraseProtected = true

	// 		// Get passphrase
	// 		passwordPrompt := promptui.Prompt{
	// 			Label: "Enter passphrase",
	// 			Mask:  '*',
	// 			Validate: func(input string) error {
	// 				if len(input) < 8 {
	// 					return errors.New("Passphrase must be at least 8 characters")
	// 				}
	// 				return nil
	// 			},
	// 		}

	// 		passphrase, err := passwordPrompt.Run()
	// 		if err != nil {
	// 			return nil, fmt.Errorf("prompt failed: %w", err)
	// 		}

	// 		// Confirm passphrase
	// 		confirmPrompt := promptui.Prompt{
	// 			Label: "Confirm passphrase",
	// 			Mask:  '*',
	// 			Validate: func(input string) error {
	// 				if input != passphrase {
	// 					return errors.New("Passphrases do not match")
	// 				}
	// 				return nil
	// 			},
	// 		}

	// 		_, err = confirmPrompt.Run()
	// 		if err != nil {
	// 			return nil, fmt.Errorf("prompt failed: %w", err)
	// 		}

	// 		configuration.Encryption. = passphrase
	// 	} else if err != promptui.ErrAbort {
	// 		return nil, fmt.Errorf("prompt failed: %w", err)
	// 	}
	// }

	// Chunking strategy prompt
	chunkStrategyPrompt := promptui.Select{
		Label: "Chunking strategy",
		Items: []string{"fixed", "cdc"},
		Templates: &promptui.SelectTemplates{
			Selected: "Chunking strategy: {{ . }}",
		},
	}

	_, chunkResult, err := chunkStrategyPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.Strategy = chunkResult

	// Chunk size prompt
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
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.ChunkSize = sizeResult

	// Compression prompt
	compressionPrompt := promptui.Select{
		Label: "Compression",
		Items: []string{"none", "gzip", "zstd"},
		Templates: &promptui.SelectTemplates{
			Selected: "Compression: {{ . }}",
		},
	}

	_, compResult, err := compressionPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Chunking.HashAlgorithm = compResult

	// Author prompt
	authorPrompt := promptui.Prompt{
		Label:     "Author",
		Default:   "nilay@dune.net",
		AllowEdit: true,
	}

	authorResult, err := authorPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Metadata.Author = authorResult

	// Tags prompt (simplified implementation)
	configuration.Metadata.Tags = []string{"research", "desert", "offline"}

	// Confirmation prompt
	confirmPrompt := promptui.Prompt{
		Label:     "Create vault now",
		IsConfirm: true,
		Default:   "y",
	}

	_, err = confirmPrompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return nil, errors.New("operation cancelled")
		}
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return configuration, nil
}
