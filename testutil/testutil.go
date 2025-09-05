// Package testutil provides common testing utilities for Sietch
package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
)

// TempDir creates a temporary directory for testing
func TempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Clean up on test completion
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("Failed to clean up temp dir %s: %v", dir, err)
		}
	})

	return dir
}

// CreateTestFile creates a test file with specified content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create directory for test file: %v", err)
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}

	return filePath
}

// CreateTestFileWithSize creates a test file with random content of specified size
func CreateTestFileWithSize(t *testing.T, dir, filename string, size int64) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create directory for test file: %v", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	defer file.Close()

	// Write random data
	written, err := io.CopyN(file, rand.Reader, size)
	if err != nil {
		t.Fatalf("Failed to write test data to %s: %v", filePath, err)
	}

	if written != size {
		t.Fatalf("Expected to write %d bytes, but wrote %d", size, written)
	}

	return filePath
}

// CreateTestVaultConfig creates a basic vault configuration for testing
func CreateTestVaultConfig(t *testing.T, vaultName string) *config.VaultConfig {
	t.Helper()

	return &config.VaultConfig{
		VaultID: "test-vault-" + vaultName,
		Name:    vaultName,
		Metadata: config.MetadataConfig{
			Author: "test-author",
			Tags:   []string{"test", "vault"},
		},
		Encryption: config.EncryptionConfig{
			Type:                "aes",
			PassphraseProtected: false,
			KeyFile:             false,
			AESConfig: &config.AESConfig{
				Mode: "gcm",
				Key:  "dGVzdC1rZXktMTIzNDU2Nzg5MGFiY2RlZg==", // base64 encoded test key
				KDF:  "pbkdf2",
			},
		},
		Chunking: config.ChunkingConfig{
			Strategy:      "fixed",
			ChunkSize:     "1MB",
			HashAlgorithm: "sha256",
		},
		Compression: "none",
		Sync: config.SyncConfig{
			Mode: "manual",
			RSA: &config.RSAConfig{
				KeySize:      2048,
				TrustedPeers: []config.TrustedPeer{},
			},
		},
	}
}

// CreateTestRSAKeyPair creates an RSA key pair for testing
func CreateTestRSAKeyPair(t *testing.T, bits int) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()

	if bits < 2048 {
		bits = 2048 // Minimum secure key size
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}

	return privateKey, &privateKey.PublicKey
}

// AssertFileExists checks if a file exists and fails the test if it doesn't
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist, but it doesn't", path)
	}
}

// AssertFileNotExists checks if a file doesn't exist and fails the test if it does
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected file %s to not exist, but it does", path)
	}
}

// AssertDirExists checks if a directory exists and fails the test if it doesn't
func AssertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Fatalf("Expected directory %s to exist, but it doesn't", path)
	}
	if err != nil {
		t.Fatalf("Error checking directory %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("Expected %s to be a directory, but it's not", path)
	}
}

// AssertFileContains checks if a file contains specific content
func AssertFileContains(t *testing.T, path, expectedContent string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), expectedContent) {
		t.Fatalf("File %s does not contain expected content '%s'", path, expectedContent)
	}
}

// AssertFileSize checks if a file has the expected size
func AssertFileSize(t *testing.T, path string, expectedSize int64) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file %s: %v", path, err)
	}

	if info.Size() != expectedSize {
		t.Fatalf("File %s has size %d, expected %d", path, info.Size(), expectedSize)
	}
}

// CaptureOutput captures stdout/stderr for testing CLI commands
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	// Create pipes for stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Replace stdout and stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Create channels to capture output
	stdoutCh := make(chan string)
	stderrCh := make(chan string)

	// Start goroutines to read from pipes
	go func() {
		defer close(stdoutCh)
		output, _ := io.ReadAll(stdoutR)
		stdoutCh <- string(output)
	}()

	go func() {
		defer close(stderrCh)
		output, _ := io.ReadAll(stderrR)
		stderrCh <- string(output)
	}()

	// Execute the function
	fn()

	// Close writers and restore original stdout/stderr
	stdoutW.Close()
	stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Get captured output
	stdout = <-stdoutCh
	stderr = <-stderrCh

	// Close readers
	stdoutR.Close()
	stderrR.Close()

	return stdout, stderr
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	if testing.Short() {
		t.Skipf("Skipping test in short mode: %s", reason)
	}
}

// CreateTestVaultStructure creates a basic vault directory structure for testing
func CreateTestVaultStructure(t *testing.T, vaultPath string) {
	t.Helper()

	// Create main vault directories
	dirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/sync",
		".sietch/chunks",
		"data",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(vaultPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create vault directory %s: %v", dirPath, err)
		}
	}
}

// GenerateTestData generates test data of specified size for benchmarking
func GenerateTestData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// CompareBytes compares two byte slices and reports differences
func CompareBytes(t *testing.T, expected, actual []byte, context string) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s: length mismatch - expected %d bytes, got %d bytes",
			context, len(expected), len(actual))
	}

	for i := 0; i < len(expected); i++ {
		if expected[i] != actual[i] {
			t.Fatalf("%s: byte mismatch at position %d - expected %02x, got %02x",
				context, i, expected[i], actual[i])
		}
	}
}

// MockConfig creates a mock configuration for testing
type MockConfig struct {
	VaultPath string
	Config    *config.VaultConfig
}

// NewMockConfig creates a new mock configuration
func NewMockConfig(t *testing.T, vaultName string) *MockConfig {
	t.Helper()

	vaultPath := TempDir(t, "mock-vault-"+vaultName)
	config := CreateTestVaultConfig(t, vaultName)

	return &MockConfig{
		VaultPath: vaultPath,
		Config:    config,
	}
}

// SetupTestVault creates a complete test vault with structure and config
func (mc *MockConfig) SetupTestVault(t *testing.T) {
	t.Helper()
	CreateTestVaultStructure(t, mc.VaultPath)

	// Create a basic vault config file
	configPath := filepath.Join(mc.VaultPath, ".sietch", "vault.yaml")
	configContent := fmt.Sprintf(`
vault_id: %s
name: %s
metadata:
  author: %s
  tags: %s
encryption:
  type: %s
  passphrase_protected: %t
chunking:
  strategy: %s
  chunk_size: %s
  hash_algorithm: %s
compression: %s
sync:
  mode: %s
`,
		mc.Config.VaultID,
		mc.Config.Name,
		mc.Config.Metadata.Author,
		strings.Join(mc.Config.Metadata.Tags, ","),
		mc.Config.Encryption.Type,
		mc.Config.Encryption.PassphraseProtected,
		mc.Config.Chunking.Strategy,
		mc.Config.Chunking.ChunkSize,
		mc.Config.Chunking.HashAlgorithm,
		mc.Config.Compression,
		mc.Config.Sync.Mode,
	)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write vault config: %v", err)
	}
}
