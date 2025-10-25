package passphrase

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/config"
	passphrasevalidation "github.com/substantialcattle5/sietch/internal/passphrase"
)

// promptForPassphrase prompts the user for a passphrase with enhanced feedback
func PromptForPassphrase(confirm bool) (string, error) {
	promptLabel := "Enter passphrase"
	if confirm {
		promptLabel = "Create new passphrase"
	}

	// Enhanced validation function with real-time feedback
	validate := func(input string) error {
		if input == "" {
			return nil // Allow empty during typing
		}

		result := passphrasevalidation.ValidateHybrid(input)

		// Show real-time feedback
		showPassphraseFeedback(input, result)

		// Only enforce validation on final submission
		if !result.Valid {
			return fmt.Errorf("%s", passphrasevalidation.GetHybridErrorMessage(result))
		}

		// Show warnings but don't prevent submission
		if len(result.Warnings) > 0 {
			fmt.Printf("\033[33m⚠️  Warning: %s\033[0m\n", result.Warnings[0])
		}

		return nil
	}

	passphrasePrompt := promptui.Prompt{
		Label:    promptLabel,
		Mask:     '*',
		Validate: validate,
	}

	passphrase, err := passphrasePrompt.Run()
	if err != nil {
		return "", fmt.Errorf("passphrase prompt failed: %w", err)
	}

	// Clear the feedback lines AND the duplicate prompt line
	linesToClear := feedbackLineCount
	if feedbackLineCount > 0 {
		linesToClear++ // Also clear the duplicate prompt line printed by promptui
	}
	for i := 0; i < linesToClear; i++ {
		fmt.Print("\033[F\033[K") // Move up and clear each line
	}

	// Show success message
	result := passphrasevalidation.ValidateHybrid(passphrase)
	if result.Valid {
		fmt.Printf("\033[32m✅ Passphrase meets all requirements (Strength: %s)\033[0m\n", result.Strength)
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

		fmt.Print("\033[32m✅ Passphrase confirmed successfully\033[0m\n")
	}

	return passphrase, nil
}

// Track feedback state to know how many lines to clear
var feedbackLineCount = 0

// showPassphraseFeedback displays real-time feedback about passphrase requirements
func showPassphraseFeedback(passphrase string, result passphrasevalidation.HybridValidationResult) {
	if passphrase == "" {
		return
	}

	// Clear previous feedback lines if they exist
	if feedbackLineCount > 0 {
		for i := 0; i < feedbackLineCount; i++ {
			fmt.Print("\033[1A\033[2K") // Move up one line and clear it
		}
		feedbackLineCount = 0
	}

	// Check individual requirements
	requirements := []struct {
		label    string
		met      bool
		progress string
	}{
		{"At least 12 characters", len(passphrase) >= 12, fmt.Sprintf("(%d/12)", len(passphrase))},
		{"Uppercase letter", hasUppercase(passphrase), ""},
		{"Lowercase letter", hasLowercase(passphrase), ""},
		{"Digit", hasDigit(passphrase), ""},
		{"Special character", hasSpecialChar(passphrase), ""},
	}

	fmt.Print("\nRequirements:\n")
	feedbackLineCount += 2

	for _, req := range requirements {
		symbol := "✗"
		color := "\033[31m" // Red
		if req.met {
			symbol = "✓"
			color = "\033[32m" // Green
		}

		label := req.label
		if req.progress != "" {
			label += " " + req.progress
		}

		fmt.Printf("   %s%s %s\033[0m\n", color, symbol, label)
		feedbackLineCount++
	}

	// Show strength meter
	score := calculateStrengthScore(passphrase, result)
	strengthLabel := getStrengthLabel(score)

	filledBars := score
	emptyBars := 10 - score

	var strengthColor string
	switch {
	case score <= 3:
		strengthColor = "\033[31m" // Red
	case score <= 6:
		strengthColor = "\033[33m" // Yellow
	case score <= 8:
		strengthColor = "\033[36m" // Cyan
	default:
		strengthColor = "\033[32m" // Green
	}

	fmt.Print("\n")
	fmt.Printf("Strength: %s%s\033[0m %s\033[37m%s\033[0m (%d/10)\n",
		strengthColor,
		strengthLabel,
		strings.Repeat("█", filledBars),
		strings.Repeat("░", emptyBars),
		score)
	feedbackLineCount += 2

	// Show warnings
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Printf("\033[33m⚠️  %s\033[0m\n", warning)
			feedbackLineCount++
		}
	}
}

// Helper functions for character validation
func hasUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func hasSpecialChar(s string) bool {
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

func calculateStrengthScore(passphrase string, result passphrasevalidation.HybridValidationResult) int {
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

func getStrengthLabel(score int) string {
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
