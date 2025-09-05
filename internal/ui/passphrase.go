/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package ui

import (
	"fmt"
	"os"
	"syscall"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/substantialcattle5/sietch/internal/config"
)

// GetPassphraseForVault retrieves the passphrase for an encrypted vault from multiple sources
// in order of preference: command-line flag, environment variable, or interactive prompt.
// It handles validation and ensures the passphrase meets security requirements.
func GetPassphraseForVault(cmd *cobra.Command, vaultConfig *config.VaultConfig) (string, error) {
	// Check if the vault needs a passphrase
	if vaultConfig.Encryption.Type == "none" || !vaultConfig.Encryption.PassphraseProtected {
		return "", nil
	}

	// Try to get passphrase from command line flag - check both flags
	passphrase := ""
	var err error

	// Try "passphrase" flag first (for backward compatibility)
	if cmd.Flags().Lookup("passphrase") != nil {
		// Only try to get string value if the flag exists and is a string
		if flag := cmd.Flags().Lookup("passphrase"); flag != nil && flag.Value.Type() == "string" {
			passphrase, err = cmd.Flags().GetString("passphrase")
			if err != nil {
				return "", fmt.Errorf("error parsing passphrase flag: %w", err)
			}
		}
	}

	// If not found, try "passphrase-value" flag
	if passphrase == "" && cmd.Flags().Lookup("passphrase-value") != nil {
		passphrase, err = cmd.Flags().GetString("passphrase-value")
		if err != nil {
			return "", fmt.Errorf("error parsing passphrase-value flag: %w", err)
		}
	}

	// If not provided as flag, check environment variable
	if passphrase == "" {
		passphrase = os.Getenv("SIETCH_PASSPHRASE")
	}

	// If still not found, prompt interactively
	if passphrase == "" {
		// Check if we should use the simple terminal prompt or promptui
		usePromptUI := false
		if cmd.Flags().Lookup("interactive") != nil {
			usePromptUI, _ = cmd.Flags().GetBool("interactive")
		}

		if usePromptUI {
			// Use promptui for interactive sessions (better UX)
			passphrasePrompt := promptui.Prompt{
				Label: "Enter encryption passphrase",
				Mask:  '*',
				Validate: func(input string) error {
					if len(input) < 8 {
						return fmt.Errorf("passphrase must be at least 8 characters")
					}
					return nil
				},
			}

			passphrase, err = passphrasePrompt.Run()
			if err != nil {
				return "", fmt.Errorf("failed to get passphrase: %w", err)
			}
		} else {
			// Use simple terminal prompt for non-interactive sessions
			fmt.Printf("Vault uses %s encryption with passphrase protection.\n", vaultConfig.Encryption.Type)
			fmt.Print("Enter passphrase: ")
			bytePassphrase, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return "", fmt.Errorf("error reading passphrase: %w", err)
			}
			fmt.Println() // Add newline after password input

			passphrase = string(bytePassphrase)

			// Validate passphrase length
			if len(passphrase) < 8 {
				return "", fmt.Errorf("passphrase must be at least 8 characters")
			}
		}
	}

	// Verify that we have a passphrase if one is required
	if passphrase == "" {
		return "", fmt.Errorf("passphrase required for encrypted vault but not provided")
	}

	return passphrase, nil
}

// GetPassphraseForInitialization retrieves the passphrase for vault encryption
// from multiple sources: command line flag, environment variable, or interactive prompt
func GetPassphraseForInitialization(cmd *cobra.Command, requireConfirmation bool) (string, error) {
	// Check if passphrase protection is enabled (we'll derive this from cmd)
	usePassphrase, err := cmd.Flags().GetBool("passphrase")
	if err != nil {
		return "", fmt.Errorf("error checking passphrase flag: %w", err)
	}

	// If passphrase protection is not enabled, return empty string
	if !usePassphrase {
		return "", nil
	}

	// Try to get passphrase from command line value flag first
	passphraseValue, err := cmd.Flags().GetString("passphrase-value")
	if err != nil {
		return "", fmt.Errorf("error parsing passphrase-value flag: %w", err)
	}

	if passphraseValue != "" {
		if len(passphraseValue) < 8 {
			return "", fmt.Errorf("passphrase must be at least 8 characters")
		}
		return passphraseValue, nil
	}

	// Check environment variable
	passphraseEnv := os.Getenv("SIETCH_PASSPHRASE")
	if passphraseEnv != "" {
		if len(passphraseEnv) < 8 {
			return "", fmt.Errorf("passphrase from environment variable must be at least 8 characters")
		}
		return passphraseEnv, nil
	}

	// Check if interactive mode is enabled
	interactiveMode, _ := cmd.Flags().GetBool("interactive")

	if interactiveMode {
		// Use promptui for better terminal UI
		passphrasePrompt := promptui.Prompt{
			Label: "Enter encryption passphrase",
			Mask:  '*',
			Validate: func(input string) error {
				if len(input) < 8 {
					return fmt.Errorf("passphrase must be at least 8 characters")
				}
				return nil
			},
		}

		enteredPassphrase, err := passphrasePrompt.Run()
		if err != nil {
			return "", fmt.Errorf("failed to get passphrase: %w", err)
		}

		// Add confirmation prompt if required
		if requireConfirmation {
			confirmPrompt := promptui.Prompt{
				Label: "Confirm passphrase",
				Mask:  '*',
				Validate: func(input string) error {
					if input != enteredPassphrase {
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

		// Removed dangerous passphrase exposure via printf
		return enteredPassphrase, nil
	} else {
		// Use simple terminal input for non-interactive mode
		fmt.Print("Enter encryption passphrase (min 8 characters): ")
		bytePassphrase, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("error reading passphrase: %w", err)
		}
		fmt.Println() // Add newline after password input

		enteredPassphrase := string(bytePassphrase)

		// Validate passphrase length
		if len(enteredPassphrase) < 8 {
			return "", fmt.Errorf("passphrase must be at least 8 characters")
		}

		// Add confirmation if required
		if requireConfirmation {
			fmt.Print("Confirm passphrase: ")
			byteConfirmation, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return "", fmt.Errorf("error reading passphrase confirmation: %w", err)
			}
			fmt.Println() // Add newline after password input

			confirmPassphrase := string(byteConfirmation)

			if enteredPassphrase != confirmPassphrase {
				return "", fmt.Errorf("passphrases do not match")
			}
		}

		return enteredPassphrase, nil
	}
}
