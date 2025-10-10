/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
