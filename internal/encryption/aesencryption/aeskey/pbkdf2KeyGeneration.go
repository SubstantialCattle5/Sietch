package aeskey

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
)

// promptPBKDF2Parameters handles configuration of PBKDF2 parameters
func PromptPBKDF2Parameters(configuration *config.VaultConfig) error {
	// PBKDF2 iterations
	iterPrompt := promptui.Select{
		Label: "PBKDF2 iterations",
		Items: []string{"100000", "200000", "500000", "1000000"},
		Templates: &promptui.SelectTemplates{
			Selected: "Iterations: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 100000: Fast, lower security
- 200000: Balanced (recommended)
- 500000: More secure, slower
- 1000000: Most secure, much slower
`,
		},
	}

	iterIdx, _, err := iterPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	iterValues := []int{100000, 200000, 500000, 1000000}
	configuration.Encryption.AESConfig.PBKDF2I = iterValues[iterIdx]

	return nil
}
