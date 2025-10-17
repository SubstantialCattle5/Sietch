package passphrase

import (
	"testing"

	passphrasevalidation "github.com/substantialcattle5/sietch/internal/passphrase"
)

// Test helper functions for character validation
func TestCharacterValidationHelpers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		upper   bool
		lower   bool
		digit   bool
		special bool
	}{
		{"empty", "", false, false, false, false},
		{"upper only", "HELLO", true, false, false, false},
		{"lower only", "hello", false, true, false, false},
		{"digit only", "12345", false, false, true, false},
		{"special only", "!@#$%", false, false, false, true},
		{"mixed", "Hello123!", true, true, true, true},
		{"no special", "Hello123", true, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasUppercase(tt.input); got != tt.upper {
				t.Errorf("hasUppercase() = %v, want %v", got, tt.upper)
			}
			if got := hasLowercase(tt.input); got != tt.lower {
				t.Errorf("hasLowercase() = %v, want %v", got, tt.lower)
			}
			if got := hasDigit(tt.input); got != tt.digit {
				t.Errorf("hasDigit() = %v, want %v", got, tt.digit)
			}
			if got := hasSpecialChar(tt.input); got != tt.special {
				t.Errorf("hasSpecialChar() = %v, want %v", got, tt.special)
			}
		})
	}
}

// Test strength calculation
func TestCalculateStrengthScore(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		result     passphrasevalidation.HybridValidationResult
		want       int
	}{
		{"empty", "", passphrasevalidation.HybridValidationResult{Score: 0}, 0},
		{"short weak", "pass", passphrasevalidation.HybridValidationResult{Score: 2, IsCommon: false}, 4},
		{"long strong", "VeryLongPassword123!", passphrasevalidation.HybridValidationResult{Score: 4, IsCommon: false}, 10},
		{"common penalty", "password", passphrasevalidation.HybridValidationResult{Score: 1, IsCommon: true}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateStrengthScore(tt.passphrase, tt.result); got != tt.want {
				t.Errorf("calculateStrengthScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test strength labels
func TestGetStrengthLabel(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{0, "Weak"},
		{3, "Weak"},
		{4, "Fair"},
		{6, "Fair"},
		{7, "Good"},
		{8, "Good"},
		{9, "Strong"},
		{10, "Strong"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := getStrengthLabel(tt.score); got != tt.want {
				t.Errorf("getStrengthLabel(%d) = %v, want %v", tt.score, got, tt.want)
			}
		})
	}
}

// Test feedback line counting (simple state test)
func TestFeedbackLineCount(t *testing.T) {
	// Reset counter
	feedbackLineCount = 0

	// Simulate some feedback display
	result := passphrasevalidation.HybridValidationResult{Valid: false, Score: 1}
	showPassphraseFeedback("test", result)

	// Should have counted some lines
	if feedbackLineCount == 0 {
		t.Error("Expected feedbackLineCount to be greater than 0 after showing feedback")
	}

	// Reset for other tests
	feedbackLineCount = 0
}

// Test empty passphrase feedback (should return early)
func TestShowPassphraseFeedbackEmpty(t *testing.T) {
	initialCount := feedbackLineCount
	result := passphrasevalidation.HybridValidationResult{}

	showPassphraseFeedback("", result)

	// Should not change the line count for empty passphrase
	if feedbackLineCount != initialCount {
		t.Errorf("Expected feedbackLineCount to remain %d, got %d", initialCount, feedbackLineCount)
	}
}
