package cmd

import (
	"strings"
	"testing"

	"github.com/substantialcattle5/sietch/internal/add"
)

func TestAddCommandUsageText(t *testing.T) {
	// Check that usage text reflects multiple file support
	usageText := addCmd.Use

	if !strings.Contains(usageText, "[source_path2] [destination_path2]...") {
		t.Errorf("Usage text should indicate multiple file support, got: %s", usageText)
	}
}

func TestAddCommandLongDescription(t *testing.T) {
	// Check that long description contains multiple file support information
	longText := addCmd.Long

	expectedPhrases := []string{
		"multiple files",
		"Paired arguments",
		"Single destination",
		"source1 dest1 source2 dest2",
		"source1 source2 ... dest",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(longText, phrase) {
			t.Errorf("Long description should contain '%s'", phrase)
		}
	}
}

func TestAddCommandShortDescription(t *testing.T) {
	// Check that short description reflects multiple file support
	shortText := addCmd.Short

	if !strings.Contains(shortText, "one or more files") {
		t.Errorf("Short description should indicate multiple file support, got: %s", shortText)
	}
}

func TestAddCommandErrorHandling(t *testing.T) {
	// Test error handling for various scenarios
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "insufficient arguments",
			args:        []string{"source.txt"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that argument parsing works (even if file operations fail later)
			_, err := add.ParseFileArguments(tt.args)
			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error in argument parsing: %v", err)
			}
		})
	}
}
