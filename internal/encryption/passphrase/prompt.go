package passphrase

import (
	"fmt"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
	passphrasevalidation "github.com/substantialcattle5/sietch/internal/passphrase"
)

// promptForPassphrase prompts the user for a passphrase
func PromptForPassphrase(confirm bool) (string, error) {
	promptLabel := "Enter passphrase"
	if confirm {
		promptLabel = "Create new passphrase"
	}

	passphrasePrompt := promptui.Prompt{
		Label: promptLabel,
		Mask:  '*',
		Validate: func(input string) error {
			result := passphrasevalidation.ValidateHybrid(input)
			if !result.Valid || len(result.Warnings) > 0 {
				return fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
			}
			return nil
		},
	}

	passphrase, err := passphrasePrompt.Run()
	if err != nil {
		return "", fmt.Errorf("passphrase prompt failed: %w", err)
	}

	if confirm {
		confirmPrompt := promptui.Prompt{
			Label: "Confirm passphrase",
			Mask:  '*',
			Validate: func(input string) error {
				if input != passphrase {
					return fmt.Errorf("passphrases do not match")
				}
				return nil
			},
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			return "", fmt.Errorf("passphrase confirmation failed: %w", err)
		}
	}

	return passphrase, nil
}

// promptPassphraseProtection asks if the vault should be protected with a passphrase
func PromptPassphraseProtection(configuration *config.VaultConfig) error {
	passphrasePrompt := promptui.Prompt{
		Label:     "Protect with passphrase",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := passphrasePrompt.Run()
	configuration.Encryption.PassphraseProtected = (err == nil)
	return nil
}
