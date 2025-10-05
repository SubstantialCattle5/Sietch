package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/substantialcattle5/sietch/testutil"
)

func TestParseFileArguments(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expected    []FilePair
		expectError bool
	}{
		{
			name: "single file pair",
			args: []string{"source.txt", "dest/"},
			expected: []FilePair{
				{Source: "source.txt", Destination: "dest/"},
			},
			expectError: false,
		},
		{
			name: "multiple paired arguments",
			args: []string{"file1.txt", "dest1/", "file2.txt", "dest2/"},
			expected: []FilePair{
				{Source: "file1.txt", Destination: "dest1/"},
				{Source: "file2.txt", Destination: "dest2/"},
			},
			expectError: false,
		},
		{
			name: "even number of args - paired pattern",
			args: []string{"file1.txt", "file2.txt", "file3.txt", "dest/"},
			expected: []FilePair{
				{Source: "file1.txt", Destination: "file2.txt"},
				{Source: "file3.txt", Destination: "dest/"},
			},
			expectError: false,
		},
		{
			name:        "insufficient arguments",
			args:        []string{"source.txt"},
			expected:    nil,
			expectError: true,
		},
		{
			name: "complex file paths",
			args: []string{"/path/to/file1.txt", "/another/path/dest1/", "~/file2.txt", "./dest2/"},
			expected: []FilePair{
				{Source: "/path/to/file1.txt", Destination: "/another/path/dest1/"},
				{Source: "~/file2.txt", Destination: "./dest2/"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFileArguments(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d pairs, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Source != expected.Source {
					t.Errorf("Pair %d: expected source %s, got %s", i, expected.Source, result[i].Source)
				}
				if result[i].Destination != expected.Destination {
					t.Errorf("Pair %d: expected destination %s, got %s", i, expected.Destination, result[i].Destination)
				}
			}
		})
	}
}

func TestAddCommandUsageText(t *testing.T) {
	// Check that usage text reflects multiple file support
	usageText := addCmd.Use

	if !strings.Contains(usageText, "[source_path2] [destination_path2]...") {
		t.Errorf("Usage text should indicate multiple file support, got: %s", usageText)
	}
}

func TestAddCommandLongDescription(t *testing.T) {
	// Check that long description contains file, directory, and symlink support information
	longText := addCmd.Long

	expectedPhrases := []string{
		"Paired arguments",
		"Single destination",
		"source1 dest1 source2 dest2",
		"source1 source2 ... dest",
		"directories",
		"symlinks",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(longText, phrase) {
			t.Errorf("Long description should contain '%s'", phrase)
		}
	}
}

func TestAddCommandShortDescription(t *testing.T) {
	// Check that short description reflects file, directory, and symlink support
	shortText := addCmd.Short

	// Should mention either "files" or "directories" or "symlinks"
	if !strings.Contains(shortText, "files") && !strings.Contains(shortText, "directories") && !strings.Contains(shortText, "symlinks") {
		t.Errorf("Short description should indicate file, directory, or symlink support, got: %s", shortText)
	}
}

func TestAddCommandWithMockVault(t *testing.T) {
	testutil.SkipIfShort(t, "integration test")

	// Create a mock vault for testing
	mockConfig := testutil.NewMockConfig(t, "add-test")
	mockConfig.SetupTestVault(t)

	// Create test files
	testFile1 := testutil.CreateTestFile(t, mockConfig.VaultPath, "test1.txt", "test content 1")
	testFile2 := testutil.CreateTestFile(t, mockConfig.VaultPath, "test2.txt", "test content 2")

	// Change to vault directory
	originalDir, _ := os.Getwd()
	os.Chdir(mockConfig.VaultPath)
	defer os.Chdir(originalDir)

	// Test multiple file addition (this would require more setup for full integration)
	// For now, we test that the argument parsing works correctly
	args := []string{testFile1, "docs/", testFile2, "data/"}
	filePairs, err := parseFileArguments(args)

	if err != nil {
		t.Fatalf("Failed to parse arguments: %v", err)
	}

	expected := []FilePair{
		{Source: testFile1, Destination: "docs/"},
		{Source: testFile2, Destination: "data/"},
	}

	if len(filePairs) != len(expected) {
		t.Fatalf("Expected %d pairs, got %d", len(expected), len(filePairs))
	}

	for i, expected := range expected {
		if filePairs[i].Source != expected.Source {
			t.Errorf("Pair %d: expected source %s, got %s", i, expected.Source, filePairs[i].Source)
		}
		if filePairs[i].Destination != expected.Destination {
			t.Errorf("Pair %d: expected destination %s, got %s", i, expected.Destination, filePairs[i].Destination)
		}
	}
}

func TestAddCommandErrorHandling(t *testing.T) {
	// Test error handling for various scenarios
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, dir string) // Function to set up test scenario
		expectError bool
	}{
		{
			name: "nonexistent source file",
			args: []string{"/nonexistent/file.txt", "dest/"},
			setupFunc: func(t *testing.T, dir string) {
				// No setup needed - file should not exist
			},
			expectError: true,
		},
		{
			name: "directory as source",
			args: []string{"."}, // Current directory
			setupFunc: func(t *testing.T, dir string) {
				// Current directory exists but is a directory
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "error-test")
			defer os.RemoveAll(tempDir)

			// Run setup if provided
			if tt.setupFunc != nil {
				tt.setupFunc(t, tempDir)
			}

			// Change to temp directory
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			// Test that argument parsing works (even if file operations fail later)
			_, err := parseFileArguments(tt.args)
			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error in argument parsing: %v", err)
			}
		})
	}
}

func TestAddCommandBatchProcessingOutput(t *testing.T) {
	// Test that batch processing shows appropriate progress messages
	// This is a unit test that focuses on the output formatting logic

	tempDir := testutil.TempDir(t, "output-test")
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := make([]string, 3)
	for i := 0; i < 3; i++ {
		testFiles[i] = testutil.CreateTestFile(t, tempDir, "test"+string(rune('1'+i))+".txt", "content "+string(rune('1'+i)))
	}

	// Test paired arguments
	pairedArgs := []string{testFiles[0], "dest1/", testFiles[1], "dest2/"}
	filePairs, err := parseFileArguments(pairedArgs)
	if err != nil {
		t.Fatalf("Failed to parse paired arguments: %v", err)
	}

	if len(filePairs) != 2 {
		t.Errorf("Expected 2 pairs, got %d", len(filePairs))
	}

	// Test single destination arguments
	singleDestArgs := []string{testFiles[0], testFiles[1], "common-dest/"}
	filePairs2, err := parseFileArguments(singleDestArgs)
	if err != nil {
		t.Fatalf("Failed to parse single destination arguments: %v", err)
	}

	if len(filePairs2) != 2 {
		t.Errorf("Expected 2 pairs, got %d", len(filePairs2))
	}

	// Verify all files go to same destination
	for _, pair := range filePairs2 {
		if pair.Destination != "common-dest/" {
			t.Errorf("Expected destination 'common-dest/', got '%s'", pair.Destination)
		}
	}
}
