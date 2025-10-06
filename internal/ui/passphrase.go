/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/substantialcattle5/sietch/internal/config"
	passphrasevalidation "github.com/substantialcattle5/sietch/internal/passphrase"
)

// readPassphraseFromStdin reads a passphrase from stdin (useful for piping)
func readPassphraseFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	passphrase, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read passphrase from stdin: %w", err)
	}
	// Remove trailing newline
	passphrase = strings.TrimRight(passphrase, "\r\n")
	return passphrase, nil
}

// readPassphraseFromFile reads a passphrase from a file
// The file should contain only the passphrase with proper permissions (0600 recommended)
func readPassphraseFromFile(filePath string) (string, error) {
	// Check file permissions for security
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to access passphrase file: %w", err)
	}

	// Warn if file permissions are too open (not strictly enforced, just a warning)
	if fileInfo.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: passphrase file has overly permissive permissions (%v). Recommended: 0600\n", fileInfo.Mode().Perm())
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}

	// Remove trailing whitespace/newlines
	passphrase := strings.TrimSpace(string(content))
	if passphrase == "" {
		return "", fmt.Errorf("passphrase file is empty")
	}

	return passphrase, nil
}

// GetPassphraseForVault retrieves the passphrase for an encrypted vault from multiple sources
// in order of preference: stdin, file, environment variable, or interactive prompt.
// It handles validation and ensures the passphrase meets security requirements.
func GetPassphraseForVault(cmd *cobra.Command, vaultConfig *config.VaultConfig) (string, error) {
	// Check if the vault needs a passphrase
	if vaultConfig.Encryption.Type == "none" || !vaultConfig.Encryption.PassphraseProtected {
		return "", nil
	}

	passphrase := ""
	var err error

	// Priority 1: Check for --passphrase-stdin flag
	if cmd.Flags().Lookup("passphrase-stdin") != nil {
		useStdin, _ := cmd.Flags().GetBool("passphrase-stdin")
		if useStdin {
			passphrase, err = readPassphraseFromStdin()
			if err != nil {
				return "", err
			}
		}
	}

	// Priority 2: Check for --passphrase-file flag
	if passphrase == "" && cmd.Flags().Lookup("passphrase-file") != nil {
		passphraseFile, _ := cmd.Flags().GetString("passphrase-file")
		if passphraseFile != "" {
			passphrase, err = readPassphraseFromFile(passphraseFile)
			if err != nil {
				return "", err
			}
		}
	}

	// Priority 3: Check environment variable
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
					result := passphrasevalidation.ValidateHybrid(input)
					if !result.Valid || len(result.Warnings) > 0 {
						return fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
					}
					return nil
				},
			}

			var err error
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

			// Validate passphrase using hybrid validation (strict rules + zxcvbn intelligence)
			result := passphrasevalidation.ValidateHybrid(passphrase)
			if !result.Valid || len(result.Warnings) > 0 {
				return "", fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
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
// from multiple sources: stdin, file, environment variable, or interactive prompt
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

	passphrase := ""

	// Priority 1: Check for --passphrase-stdin flag
	if cmd.Flags().Lookup("passphrase-stdin") != nil {
		useStdin, _ := cmd.Flags().GetBool("passphrase-stdin")
		if useStdin {
			passphrase, err = readPassphraseFromStdin()
			if err != nil {
				return "", err
			}
			// Validate the passphrase
			result := passphrasevalidation.ValidateHybrid(passphrase)
			if !result.Valid || len(result.Warnings) > 0 {
				return "", fmt.Errorf("passphrase from stdin: %s", passphrasevalidation.GetHybridErrorMessage(result))
			}
			return passphrase, nil
		}
	}

	// Priority 2: Check for --passphrase-file flag
	if cmd.Flags().Lookup("passphrase-file") != nil {
		passphraseFile, _ := cmd.Flags().GetString("passphrase-file")
		if passphraseFile != "" {
			passphrase, err = readPassphraseFromFile(passphraseFile)
			if err != nil {
				return "", err
			}
			// Validate the passphrase
			result := passphrasevalidation.ValidateHybrid(passphrase)
			if !result.Valid || len(result.Warnings) > 0 {
				return "", fmt.Errorf("passphrase from file: %s", passphrasevalidation.GetHybridErrorMessage(result))
			}
			return passphrase, nil
		}
	}

	// Priority 3: Check environment variable
	passphraseEnv := os.Getenv("SIETCH_PASSPHRASE")
	if passphraseEnv != "" {
		result := passphrasevalidation.ValidateHybrid(passphraseEnv)
		if !result.Valid || len(result.Warnings) > 0 {
			return "", fmt.Errorf("passphrase from environment variable: %s", passphrasevalidation.GetHybridErrorMessage(result))
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
				result := passphrasevalidation.ValidateHybrid(input)
				if !result.Valid || len(result.Warnings) > 0 {
					return fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
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

		// Validate passphrase using hybrid validation (strict rules + zxcvbn intelligence)
		result := passphrasevalidation.ValidateHybrid(enteredPassphrase)
		if !result.Valid || len(result.Warnings) > 0 {
			return "", fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
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
