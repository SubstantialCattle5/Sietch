// Package e2e provides end-to-end testing utilities for Sietch CLI
package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVault represents a temporary vault for E2E testing
type TestVault struct {
	Path       string
	t          *testing.T
	binaryPath string
}

// NewTestVault creates a new temporary vault for testing
func NewTestVault(t *testing.T) *TestVault {
	t.Helper()

	// Create temp directory
	vaultPath := t.TempDir()

	// Get binary path (build if needed)
	binaryPath := ensureBinary(t)

	return &TestVault{
		Path:       vaultPath,
		t:          t,
		binaryPath: binaryPath,
	}
}

// ensureBinary builds the sietch binary if it doesn't exist and returns its path
func ensureBinary(t *testing.T) string {
	t.Helper()

	// Check if binary already exists in project root
	projectRoot := getProjectRoot(t)
	binaryPath := filepath.Join(projectRoot, "sietch")

	// Check if binary exists and is recent
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Build the binary
	t.Logf("Building sietch binary...")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build sietch binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

// getProjectRoot finds the project root directory
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory and walk up to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// Init initializes a new vault
func (v *TestVault) Init(t *testing.T, extraArgs ...string) (string, string, error) {
	t.Helper()

	args := []string{"init", "--name", "test-vault"}
	args = append(args, extraArgs...)

	return v.RunCommand(t, args...)
}

// Add adds a file to the vault
func (v *TestVault) Add(t *testing.T, sourcePath, destination string, extraArgs ...string) (string, string, error) {
	t.Helper()

	args := []string{"add", sourcePath, destination}
	args = append(args, extraArgs...)

	return v.RunCommand(t, args...)
}

// Ls lists files in the vault
func (v *TestVault) Ls(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	lsArgs := []string{"ls"}
	lsArgs = append(lsArgs, args...)

	return v.RunCommand(t, lsArgs...)
}

// RunCommand runs a sietch command in the vault directory
func (v *TestVault) RunCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := exec.Command(v.binaryPath, args...)
	cmd.Dir = v.Path

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	// Log command execution for debugging
	if t.Failed() || testing.Verbose() {
		t.Logf("Command: sietch %s", strings.Join(args, " "))
		t.Logf("Working Dir: %s", v.Path)
		t.Logf("Exit Code: %v", err)
		if stdout != "" {
			t.Logf("Stdout:\n%s", stdout)
		}
		if stderr != "" {
			t.Logf("Stderr:\n%s", stderr)
		}
	}

	return stdout, stderr, err
}

// CreateFile creates a test file in the vault directory
func (v *TestVault) CreateFile(t *testing.T, relativePath, content string) string {
	t.Helper()

	fullPath := filepath.Join(v.Path, relativePath)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directories for %s: %v", relativePath, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", relativePath, err)
	}

	return fullPath
}

// CreateFileWithSize creates a test file with specified size
func (v *TestVault) CreateFileWithSize(t *testing.T, relativePath string, size int64) string {
	t.Helper()

	fullPath := filepath.Join(v.Path, relativePath)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directories for %s: %v", relativePath, err)
	}

	// Create file with repeated content to reach desired size
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("Failed to create file %s: %v", relativePath, err)
	}
	defer f.Close()

	// Write in chunks to avoid memory issues
	chunk := []byte(strings.Repeat("A", 1024))
	remaining := size
	for remaining > 0 {
		toWrite := int64(len(chunk))
		if toWrite > remaining {
			toWrite = remaining
		}
		if _, err := f.Write(chunk[:toWrite]); err != nil {
			t.Fatalf("Failed to write to file %s: %v", relativePath, err)
		}
		remaining -= toWrite
	}

	return fullPath
}

// AssertOutputContains checks if output contains expected string
func AssertOutputContains(t *testing.T, output, expected, context string) {
	t.Helper()

	if !strings.Contains(output, expected) {
		t.Errorf("%s: output does not contain expected string.\nExpected substring: %q\nActual output:\n%s",
			context, expected, output)
	}
}

// AssertOutputNotContains checks if output does not contain a string
func AssertOutputNotContains(t *testing.T, output, unexpected, context string) {
	t.Helper()

	if strings.Contains(output, unexpected) {
		t.Errorf("%s: output contains unexpected string.\nUnexpected substring: %q\nActual output:\n%s",
			context, unexpected, output)
	}
}

// AssertCommandSuccess checks if command succeeded
func AssertCommandSuccess(t *testing.T, err error, stderr, context string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: command failed: %v\nStderr: %s", context, err, stderr)
	}
}

// AssertCommandFails checks if command failed as expected
func AssertCommandFails(t *testing.T, err error, context string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected command to fail, but it succeeded", context)
	}
}

// AssertFileCount checks the number of files listed in output
func AssertFileCount(t *testing.T, output string, expectedCount int, context string) {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Filter out empty lines and headers (lines starting with SIZE, MODIFIED, etc.)
	fileLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip header lines
		if strings.HasPrefix(trimmed, "SIZE") ||
			strings.HasPrefix(trimmed, "shared_chunks:") ||
			strings.HasPrefix(trimmed, "No files") {
			continue
		}
		fileLines++
	}

	if fileLines != expectedCount {
		t.Errorf("%s: expected %d files, got %d files.\nOutput:\n%s",
			context, expectedCount, fileLines, output)
	}
}

// InitializeVault is a helper that creates and initializes a test vault
func InitializeVault(t *testing.T) *TestVault {
	t.Helper()

	vault := NewTestVault(t)

	// Initialize vault
	_, stderr, err := vault.Init(t)
	AssertCommandSuccess(t, err, stderr, "vault initialization")

	// The 'init' command creates a subdirectory; update the path to point to it
	vault.Path = filepath.Join(vault.Path, "test-vault")

	return vault
}

// helpers.go

// SetupVaultWithFiles creates a vault and adds sample files
func SetupVaultWithFiles(t *testing.T, files map[string]string) *TestVault {
	t.Helper()

	// This function correctly creates, initializes, and sets the path for the vault.
	vault := InitializeVault(t)

	// Create and add each file from the input map.
	for filename, content := range files {
		// 1. Create the file directly inside the vault's working directory.
		//    This is the crucial step that was missing.
		vault.CreateFile(t, filename, content)

		// 2. Now that the file exists, add it to the vault's internal storage.
		//    This command will now succeed.
		_, stderr, err := vault.Add(t, filename, "test/")
		AssertCommandSuccess(t, err, stderr, fmt.Sprintf("adding file %s", filename))
	}

	return vault
}