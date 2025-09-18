package vault

import (
	"errors"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// promptBasicConfig asks for basic vault configuration
func PromptBasicConfig(configuration *config.VaultConfig) error {
	namePrompt := promptui.Prompt{
		Label:     constants.VaultNameLabel,
		Default:   constants.VaultNameDefault,
		AllowEdit: true,
		Validate: func(input string) error {
			if len(input) < constants.VaultNameMinLength {
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

// promptMetadataConfig asks for metadata information
func PromptMetadataConfig(configuration *config.VaultConfig) error {
	// Author prompt
	authorPrompt := promptui.Prompt{
		Label:     constants.AuthorLabel,
		Default:   constants.AuthorDefault,
		AllowEdit: constants.AuthorAllowEdit,
	}

	authorResult, err := authorPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Metadata.Author = authorResult

	// Tags prompt - allow multiple tags
	tagsPrompt := promptui.Prompt{
		Label:     constants.TagsLabel,
		Default:   constants.TagsDefault,
		AllowEdit: constants.TagsAllowEdit,
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
