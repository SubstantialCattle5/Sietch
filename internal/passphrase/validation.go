package passphrase

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidationResult contains validation results and user feedback
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// Validate validates a passphrase against security requirements
// Following NIST and OWASP best practices:
// - Minimum 12 characters (stronger than previous 8)
// - At least 1 uppercase letter
// - At least 1 lowercase letter
// - At least 1 digit
// - At least 1 special character
func Validate(passphrase string) ValidationResult {
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

// GetErrorMessage returns a user-friendly error message
func GetErrorMessage(result ValidationResult) string {
	if result.Valid {
		return ""
	}

	if len(result.Errors) == 1 {
		return result.Errors[0]
	}

	return fmt.Sprintf("Passphrase requirements not met:\n• %s", strings.Join(result.Errors, "\n• "))
}

// GetStrength provides simple strength assessment
func GetStrength(passphrase string) string {
	if len(passphrase) < 8 {
		return "Very Weak"
	}

	result := Validate(passphrase)
	if !result.Valid {
		return "Weak"
	}

	// Count character types
	types := 0
	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false

	for _, char := range passphrase {
		if unicode.IsUpper(char) && !hasUpper {
			hasUpper = true
			types++
		}
		if unicode.IsLower(char) && !hasLower {
			hasLower = true
			types++
		}
		if unicode.IsDigit(char) && !hasDigit {
			hasDigit = true
			types++
		}
		if isSpecialChar(char) && !hasSpecial {
			hasSpecial = true
			types++
		}
	}

	// Simple strength scoring based on length and character diversity
	if len(passphrase) >= 16 && types == 4 {
		return "Strong"
	} else if len(passphrase) >= 12 && types >= 3 {
		return "Medium"
	} else {
		return "Weak"
	}
}
