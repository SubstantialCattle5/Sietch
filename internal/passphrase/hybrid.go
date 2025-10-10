package passphrase

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/nbutton23/zxcvbn-go"
)

// HybridValidationResult combines our strict rules with zxcvbn intelligence
type HybridValidationResult struct {
	Valid     bool
	Errors    []string
	Warnings  []string
	Strength  string
	Score     int
	CrackTime string
	IsCommon  bool
}

// ValidateHybrid provides the best of both approaches:
// 1. Our strict character requirements (non-negotiable)
// 2. zxcvbn's intelligence for common password detection
func ValidateHybrid(passphrase string) HybridValidationResult {
	result := HybridValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// FIRST: Apply our strict requirements (non-negotiable)
	basicResult := validateBasic(passphrase)
	result.Valid = basicResult.Valid
	result.Errors = append(result.Errors, basicResult.Errors...)

	// SECOND: Only run expensive zxcvbn check if basic requirements are met
	if result.Valid {
		zxcvbnResult := zxcvbn.PasswordStrength(passphrase, nil)
		result.Score = zxcvbnResult.Score
		result.CrackTime = formatCrackTime(zxcvbnResult.CrackTime)

		// Detect common/predictable passwords
		if zxcvbnResult.Score <= 1 {
			result.IsCommon = true
			result.Warnings = append(result.Warnings,
				"This passphrase is predictable or commonly used. Consider making it more unique.")
		}

		// Enhanced strength assessment based on zxcvbn score
		if zxcvbnResult.Score >= 3 {
			result.Strength = "Strong"
		} else if zxcvbnResult.Score >= 2 {
			result.Strength = "Good"
		} else {
			result.Strength = "Fair"
		}
	} else {
		// If it doesn't pass basic validation, use our basic strength assessment
		// No need to run expensive zxcvbn on invalid passwords
		result.Strength = GetStrength(passphrase)
		result.Score = 0 // Invalid passwords get minimum score
	}

	return result
}

// GetHybridErrorMessage returns user-friendly error message
func GetHybridErrorMessage(result HybridValidationResult) string {
	if result.Valid && len(result.Warnings) == 0 {
		return ""
	}

	var messages []string

	// Critical errors first
	messages = append(messages, result.Errors...)

	// Then warnings
	for _, warning := range result.Warnings {
		messages = append(messages, "⚠️  "+warning)
	}

	if len(messages) == 1 {
		return messages[0]
	}

	return fmt.Sprintf("Passphrase feedback:\n• %s", strings.Join(messages, "\n• "))
}

// validateBasic performs the basic character and length validation
func validateBasic(passphrase string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Minimum length check
	if len(passphrase) < 12 {
		result.Valid = false
		result.Errors = append(result.Errors, "passphrase must be at least 12 characters long")
	}

	// Character set requirements
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false

	for _, char := range passphrase {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case isSpecialChar(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		result.Valid = false
		result.Errors = append(result.Errors, "passphrase must contain at least one uppercase letter")
	}
	if !hasLower {
		result.Valid = false
		result.Errors = append(result.Errors, "passphrase must contain at least one lowercase letter")
	}
	if !hasDigit {
		result.Valid = false
		result.Errors = append(result.Errors, "passphrase must contain at least one digit")
	}
	if !hasSpecial {
		result.Valid = false
		result.Errors = append(result.Errors, "passphrase must contain at least one special character (!@#$%^&*()_+-=[]{}|;:,.<>?)")
	}

	return result
}

// isSpecialChar checks if a character is a special character
func isSpecialChar(char rune) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	return strings.ContainsRune(specialChars, char)
}

// formatCrackTime formats the crack time display for user-friendly output
func formatCrackTime(crackTime interface{}) string {
	if crackTime == nil {
		return "unknown"
	}
	return fmt.Sprintf("%v", crackTime)
}
