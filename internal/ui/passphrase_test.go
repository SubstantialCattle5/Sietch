/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	passphrasevalidation "github.com/substantialcattle5/sietch/internal/passphrase"
)

func TestReadPassphraseFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		permissions os.FileMode
		wantErr     bool
		expected    string
		errContains string
	}{
		{
			name:        "valid passphrase file with secure permissions",
			content:     "MySecureP@ssw0rd!",
			permissions: 0600,
			wantErr:     false,
			expected:    "MySecureP@ssw0rd!",
		},
		{
			name:        "passphrase with trailing newline",
			content:     "MySecureP@ssw0rd!\n",
			permissions: 0600,
			wantErr:     false,
			expected:    "MySecureP@ssw0rd!",
		},
		{
			name:        "passphrase with trailing spaces",
			content:     "MySecureP@ssw0rd!   \n",
			permissions: 0600,
			wantErr:     false,
			expected:    "MySecureP@ssw0rd!",
		},
		{
			name:        "empty file",
			content:     "",
			permissions: 0600,
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "file with only whitespace",
			content:     "   \n  \n",
			permissions: 0600,
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "file with insecure permissions (warning only)",
			content:     "MySecureP@ssw0rd!",
			permissions: 0644,
			wantErr:     false,
			expected:    "MySecureP@ssw0rd!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "passphrase.txt")

			// Write content to file
			err := os.WriteFile(tmpFile, []byte(tt.content), tt.permissions)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Read the passphrase
			result, err := readPassphraseFromFile(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected passphrase %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReadPassphraseFromFileErrors(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := readPassphraseFromFile("/nonexistent/path/to/file")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := readPassphraseFromFile(tmpDir)
		if err == nil {
			t.Error("Expected error when passing directory")
		}
	})
}

func TestReadPassphraseFromStdin(t *testing.T) {
	// Note: Testing stdin is tricky because it requires actual stdin redirection
	// This test demonstrates the concept but may need to be run manually
	t.Skip("Skipping stdin test - requires manual testing with actual stdin redirection")
}

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
			if got := hasUppercaseChar(tt.input); got != tt.upper {
				t.Errorf("hasUppercaseChar() = %v, want %v", got, tt.upper)
			}
			if got := hasLowercaseChar(tt.input); got != tt.lower {
				t.Errorf("hasLowercaseChar() = %v, want %v", got, tt.lower)
			}
			if got := hasDigitChar(tt.input); got != tt.digit {
				t.Errorf("hasDigitChar() = %v, want %v", got, tt.digit)
			}
			if got := hasSpecialCharacter(tt.input); got != tt.special {
				t.Errorf("hasSpecialCharacter() = %v, want %v", got, tt.special)
			}
		})
	}
}

// Test strength calculation
func TestCalculatePassphraseStrength(t *testing.T) {
	result := passphrasevalidation.HybridValidationResult{Score: 2, IsCommon: false}

	tests := []struct {
		name       string
		passphrase string
		result     passphrasevalidation.HybridValidationResult
		want       int
	}{
		{"empty", "", result, 0},
		{"short weak", "pass", result, 4}, // score 2*2 = 4
		{"long strong", "VeryLongPassword123!", passphrasevalidation.HybridValidationResult{Score: 4, IsCommon: false}, 10}, // 4*2 + 1 + 1 = 10
		{"common penalty", "password", passphrasevalidation.HybridValidationResult{Score: 1, IsCommon: true}, 0},            // 1*2 - 2 = 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculatePassphraseStrength(tt.passphrase, tt.result); got != tt.want {
				t.Errorf("calculatePassphraseStrength() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test strength labels
func TestGetPassphraseStrengthLabel(t *testing.T) {
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
			if got := getPassphraseStrengthLabel(tt.score); got != tt.want {
				t.Errorf("getPassphraseStrengthLabel(%d) = %v, want %v", tt.score, got, tt.want)
			}
		})
	}
}
