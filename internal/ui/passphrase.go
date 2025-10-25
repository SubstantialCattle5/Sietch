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
		// Use enhanced passphrase prompt with real-time feedback
		fmt.Println()
		fmt.Println("🔐 Enhanced Passphrase Entry")
		fmt.Println("-" + strings.Repeat("=", 28))
		return getEnhancedPassphrase(requireConfirmation)
	} else {
		// Use simple terminal input for non-interactive mode
		// Show all requirements upfront
		fmt.Println("\nPassphrase Requirements:")
		fmt.Println("  • At least 12 characters")
		fmt.Println("  • Uppercase letter")
		fmt.Println("  • Lowercase letter")
		fmt.Println("  • Digit")
		fmt.Println("  • Special character (!@#$%^&*()_+-=[]{}|;:,.<>?)")
		fmt.Println()
		fmt.Print("Enter encryption passphrase: ")
		bytePassphrase, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("error reading passphrase: %w", err)
		}
		fmt.Println() // Add newline after password input

		enteredPassphrase := string(bytePassphrase)

		// Validate passphrase using hybrid validation (strict rules + zxcvbn intelligence)
		result := passphrasevalidation.ValidateHybrid(enteredPassphrase)
		if !result.Valid {
			return "", fmt.Errorf("passphrase does not meet the requirements listed above")
		}

		// Show warnings but don't fail
		if len(result.Warnings) > 0 {
			fmt.Printf("\033[33m⚠️  Warning: %s\033[0m\n", result.Warnings[0])
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

// getEnhancedPassphrase provides an enhanced passphrase prompt with real-time feedback
func getEnhancedPassphrase(requireConfirmation bool) (string, error) {
	// Show initial instructions
	fmt.Println("💡 Passphrase requirements:")
	fmt.Println("   • At least 12 characters")
	fmt.Println("   • Uppercase letter (A-Z)")
	fmt.Println("   • Lowercase letter (a-z)")
	fmt.Println("   • Digit (0-9)")
	fmt.Println("   • Special character (!@#$%^&*()_+-=[]{}|;:,.<>?)")
	fmt.Println()

	// Use custom input with true in-place real-time feedback
	passphrase, err := getPassphraseWithInPlaceFeedback("Enter encryption passphrase")
	if err != nil {
		return "", fmt.Errorf("passphrase prompt failed: %w", err)
	}

	// Show success message
	result := passphrasevalidation.ValidateHybrid(passphrase)
	if result.Valid {
		fmt.Printf("✅ Passphrase meets all requirements (Strength: %s)\n", result.Strength)
	}

	// Handle confirmation if required
	if requireConfirmation {
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

		fmt.Println("✅ Passphrase confirmed successfully")
	}

	return passphrase, nil
}

// Helper functions for character validation
func hasUppercaseChar(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowercaseChar(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func hasDigitChar(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func hasSpecialCharacter(s string) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, r := range s {
		for _, special := range specialChars {
			if r == special {
				return true
			}
		}
	}
	return false
}

func calculatePassphraseStrength(passphrase string, result passphrasevalidation.HybridValidationResult) int {
	if len(passphrase) == 0 {
		return 0
	}

	// Base score from zxcvbn (0-4) converted to 0-10 scale
	score := result.Score * 2

	// Bonus points for meeting basic requirements
	if len(passphrase) >= 12 {
		score += 1
	}
	if len(passphrase) >= 16 {
		score += 1
	}

	// Penalty for common passwords
	if result.IsCommon {
		score -= 2
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	return score
}

func getPassphraseStrengthLabel(score int) string {
	switch {
	case score <= 3:
		return "Weak"
	case score <= 6:
		return "Fair"
	case score <= 8:
		return "Good"
	default:
		return "Strong"
	}
}

// getPassphraseWithInPlaceFeedback implements true in-place real-time feedback
func getPassphraseWithInPlaceFeedback(label string) (string, error) {
	fmt.Printf("%s: ", label)

	// Initialize feedback area - show it once and then only update it
	fmt.Print("\n\nStatus: ✗12+ ✗Upper ✗Lower ✗Digit ✗Special | Strength: Weak ░░░░░ (0/10)\n")
	fmt.Printf("\033[1A\033[%dC", len(label)+2) // Move cursor back to input position

	var passphrase []rune

	// Set terminal to raw mode for character-by-character input
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	for {
		// Read single character
		var buf [3]byte // UTF-8 can be up to 3 bytes
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return "", err
		}

		if n == 1 {
			char := buf[0]

			switch char {
			case 13: // Enter key
				currentPass := string(passphrase)
				result := passphrasevalidation.ValidateHybrid(currentPass)
				if result.Valid {
					// Move to next line and show success
					fmt.Print("\n\n")
					fmt.Printf("✅ Passphrase accepted (Strength: %s)\n", result.Strength)
					return currentPass, nil
				} else {
					// Beep and stay in place (password invalid)
					fmt.Print("\a") // Bell sound
				}

			case 3: // Ctrl+C
				fmt.Print("\n")
				return "", fmt.Errorf("^C")

			case 127, 8: // Backspace/Delete
				if len(passphrase) > 0 {
					passphrase = passphrase[:len(passphrase)-1]
					fmt.Print("\b \b") // Erase character visually
					updateStatusLine(string(passphrase))
				}

			default:
				if char >= 32 && char <= 126 { // Printable ASCII
					passphrase = append(passphrase, rune(char))
					fmt.Print("*")
					updateStatusLine(string(passphrase))
				}
			}
		}
	}
}

// updateStatusLine updates just the status line in place
func updateStatusLine(passphrase string) {
	if len(passphrase) == 0 {
		// Reset to initial state
		fmt.Print("\033[s")  // Save cursor position
		fmt.Print("\033[1B") // Move down one line to status line
		fmt.Print("\033[2K") // Clear entire line
		fmt.Print("Status: ✗12+ ✗Upper ✗Lower ✗Digit ✗Special | Strength: Weak ░░░░░ (0/10)")
		fmt.Print("\033[u") // Restore cursor position
		return
	}

	// Save current cursor position
	fmt.Print("\033[s")

	// Move to status line (one line down)
	fmt.Print("\033[1B")
	fmt.Print("\033[2K") // Clear the line

	// Check requirements
	hasLength := len(passphrase) >= 12
	hasUpper := hasUppercaseChar(passphrase)
	hasLower := hasLowercaseChar(passphrase)
	hasDigit := hasDigitChar(passphrase)
	hasSpecial := hasSpecialCharacter(passphrase)

	// Build status with colors
	statusParts := []string{
		fmt.Sprintf("%s12+", getSymbol(hasLength)),
		fmt.Sprintf("%sUpper", getSymbol(hasUpper)),
		fmt.Sprintf("%sLower", getSymbol(hasLower)),
		fmt.Sprintf("%sDigit", getSymbol(hasDigit)),
		fmt.Sprintf("%sSpecial", getSymbol(hasSpecial)),
	}

	if hasLength {
		statusParts[0] = fmt.Sprintf("%s12+(%d)", getSymbol(hasLength), len(passphrase))
	}

	// Calculate strength
	result := passphrasevalidation.ValidateHybrid(passphrase)
	score := calculatePassphraseStrength(passphrase, result)
	strengthLabel := getPassphraseStrengthLabel(score)

	// Create compact strength meter (5 bars)
	filledBars := (score + 1) / 2 // 0-10 -> 0-5
	if filledBars > 5 {
		filledBars = 5
	}
	emptyBars := 5 - filledBars

	meter := strings.Repeat("█", filledBars) + strings.Repeat("░", emptyBars)

	// Write the complete status line
	fmt.Printf("Status: %s | Strength: %s %s (%d/10)",
		strings.Join(statusParts, " "),
		strengthLabel,
		meter,
		score)

	// Add warning if common password
	if result.IsCommon {
		fmt.Print(" | ⚠️ Common")
	}

	// Restore cursor position
	fmt.Print("\033[u")
}

// getSymbol returns colored checkmark or X
func getSymbol(met bool) string {
	if met {
		return "\033[32m✓\033[0m" // Green checkmark
	}
	return "\033[31m✗\033[0m" // Red X
}
