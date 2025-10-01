package deduplication

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
)

// PromptDeduplicationConfig asks for deduplication settings interactively
func PromptDeduplicationConfig(configuration *config.VaultConfig) error {
	// Enabled prompt
	enabledPrompt := promptui.Select{
		Label: "Enable deduplication",
		Items: []string{"yes", "no"},
		Templates: &promptui.SelectTemplates{
			Selected: "Deduplication enabled: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "yes" }}Enable content-based deduplication to save storage space
{{ else }}Disable deduplication (files will not share chunks){{ end }}
`,
		},
	}

	_, enabledResult, err := enabledPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Deduplication.Enabled = (enabledResult == "yes")

	// If deduplication is disabled, set defaults and return
	if !configuration.Deduplication.Enabled {
		configuration.Deduplication.Strategy = ""
		configuration.Deduplication.MinChunkSize = ""
		configuration.Deduplication.MaxChunkSize = ""
		configuration.Deduplication.GCThreshold = 0
		configuration.Deduplication.IndexEnabled = false
		return nil
	}

	// Strategy prompt
	strategyPrompt := promptui.Select{
		Label: "Deduplication strategy",
		Items: []string{"content", "fingerprint"},
		Templates: &promptui.SelectTemplates{
			Selected: "Strategy: {{ . }}",
			Active:   "▸ {{ . }} {{ if eq . \"content\" }}(recommended){{ end }}",
			Inactive: "  {{ . }} {{ if eq . \"content\" }}(recommended){{ end }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "content" }}Content-based deduplication using full chunk hashing (recommended)
{{ else if eq . "fingerprint" }}Fingerprint-based deduplication using sampling (faster but less accurate){{ end }}
`,
		},
	}

	_, strategyResult, err := strategyPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Deduplication.Strategy = strategyResult

	// Min chunk size prompt
	minSizePrompt := promptui.Prompt{
		Label:   "Minimum chunk size",
		Default: "1KB",
		Validate: func(input string) error {
			if len(input) < 1 {
				return errors.New("size must not be empty")
			}
			return nil
		},
	}

	minSizeResult, err := minSizePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Deduplication.MinChunkSize = minSizeResult

	// Max chunk size prompt
	maxSizePrompt := promptui.Prompt{
		Label:   "Maximum chunk size",
		Default: "64MB",
		Validate: func(input string) error {
			if len(input) < 1 {
				return errors.New("size must not be empty")
			}
			return nil
		},
	}

	maxSizeResult, err := maxSizePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Deduplication.MaxChunkSize = maxSizeResult

	// GC threshold prompt
	gcThresholdPrompt := promptui.Prompt{
		Label:   "Garbage collection threshold (number of unreferenced chunks before GC is suggested)",
		Default: "1000",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil {
				return errors.New("must be a valid number")
			}
			if val < 0 {
				return errors.New("must be non-negative")
			}
			return nil
		},
	}

	gcThresholdResult, err := gcThresholdPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	gcThreshold, err := strconv.Atoi(gcThresholdResult)
	if err != nil {
		return fmt.Errorf("invalid GC threshold: %w", err)
	}
	configuration.Deduplication.GCThreshold = gcThreshold

	// Index enabled prompt
	indexEnabledPrompt := promptui.Select{
		Label: "Enable deduplication index",
		Items: []string{"yes", "no"},
		Templates: &promptui.SelectTemplates{
			Selected: "Index enabled: {{ . }}",
			Active:   "▸ {{ . }} {{ if eq . \"yes\" }}(recommended){{ end }}",
			Inactive: "  {{ . }} {{ if eq . \"yes\" }}(recommended){{ end }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "yes" }}Enable chunk index for faster lookups (recommended for better performance)
{{ else }}Disable index (slower lookups but uses less memory){{ end }}
`,
		},
	}

	_, indexEnabledResult, err := indexEnabledPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Deduplication.IndexEnabled = (indexEnabledResult == "yes")

	return nil
}
