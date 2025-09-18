package aeskey

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// PromptScryptParameters handles configuration of scrypt parameters
func PromptScryptParameters(configuration *config.VaultConfig) error {
	advancedPrompt := promptui.Prompt{
		Label:     "Configure advanced scrypt parameters",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := advancedPrompt.Run()
	if err == nil { // User selected yes
		return PromptAdvancedScryptParameters(configuration)
	}
	// Default scrypt parameters
	configuration.Encryption.AESConfig.ScryptN = constants.DefaultScryptN
	configuration.Encryption.AESConfig.ScryptR = constants.DefaultScryptR
	configuration.Encryption.AESConfig.ScryptP = constants.DefaultScryptP
	return nil
}

// promptAdvancedScryptParameters handles configuration of advanced scrypt parameters
func PromptAdvancedScryptParameters(configuration *config.VaultConfig) error {
	// Scrypt N parameter
	nPrompt := promptui.Select{
		Label: "Scrypt N parameter (CPU/memory cost)",
		Items: []string{"16384", "32768", "65536", "131072"},
		Templates: &promptui.SelectTemplates{
			Selected: "N: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 16384: Fast, lower security
- 32768: Balanced (recommended)
- 65536: More secure, slower
- 131072: Most secure, much slower
`,
		},
	}

	nIdx, _, err := nPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	nValues := []int{16384, 32768, 65536, 131072}
	configuration.Encryption.AESConfig.ScryptN = nValues[nIdx]

	// Scrypt r and p parameters
	configuration.Encryption.AESConfig.ScryptR = 8
	configuration.Encryption.AESConfig.ScryptP = 1

	return nil
}
