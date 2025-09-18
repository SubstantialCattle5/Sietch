package validation

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/testutil"
)

func TestHandleKeyGeneration(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) (string, KeyGenParams)
		wantErr        bool
		errContains    string
		validateResult func(t *testing.T, result *config.KeyConfig, vaultPath string)
	}{
		{
			name: "generate new AES key with passphrase",
			setupFunc: func(t *testing.T) (string, KeyGenParams) {
				vaultPath := testutil.TempDir(t, "vault-aes-passphrase")
				testutil.CreateTestVaultStructure(t, vaultPath)
				params := KeyGenParams{
					KeyType:          "aes",
					UsePassphrase:    true,
					AESMode:          "gcm",
					UseScrypt:        true,
					ScryptN:          32768,
					ScryptR:          8,
					ScryptP:          1,
					PBKDF2Iterations: 10000,
				}
				return vaultPath, params
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *config.KeyConfig, vaultPath string) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned, got nil")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.Key == "" {
					t.Error("Expected key to be generated")
				}
				if result.Salt == "" {
					t.Error("Expected salt to be generated for passphrase-protected key")
				}
			},
		},
		{
			name: "generate new AES key without passphrase",
			setupFunc: func(t *testing.T) (string, KeyGenParams) {
				vaultPath := testutil.TempDir(t, "vault-aes-no-passphrase")
				testutil.CreateTestVaultStructure(t, vaultPath)
				params := KeyGenParams{
					KeyType:       "aes",
					UsePassphrase: false,
					AESMode:       "gcm",
				}
				return vaultPath, params
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *config.KeyConfig, vaultPath string) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned, got nil")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.Key == "" {
					t.Error("Expected key to be generated")
				}
				if result.Salt != "" {
					t.Error("Expected no salt for non-passphrase-protected key")
				}
			},
		},
		{
			name: "import key from file",
			setupFunc: func(t *testing.T) (string, KeyGenParams) {
				vaultPath := testutil.TempDir(t, "vault-import-key")
				testutil.CreateTestVaultStructure(t, vaultPath)

				// Create a test key file
				keyFile := testutil.CreateTestFile(t, vaultPath, "test.key", "1234567890123456789012345678901234567890") // 40 chars, will be truncated to 32 for AES-256

				params := KeyGenParams{
					KeyType: "aes",
					KeyFile: keyFile,
				}
				return vaultPath, params
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *config.KeyConfig, vaultPath string) {
				keyPath := filepath.Join(vaultPath, ".sietch", "keys", "secret.key")
				if _, err := os.Stat(keyPath); os.IsNotExist(err) {
					t.Error("Expected key file to be created")
				}

				// Check that the key was imported correctly
				keyData, err := os.ReadFile(keyPath)
				if err != nil {
					t.Fatalf("Failed to read imported key: %v", err)
				}
				expectedKey := "1234567890123456789012345678901234567890"
				if string(keyData) != expectedKey {
					t.Errorf("Expected imported key %q, got %q", expectedKey, string(keyData))
				}
			},
		},
		{
			name: "generate AES key with CBC mode",
			setupFunc: func(t *testing.T) (string, KeyGenParams) {
				vaultPath := testutil.TempDir(t, "vault-aes-cbc")
				testutil.CreateTestVaultStructure(t, vaultPath)
				params := KeyGenParams{
					KeyType:       "aes",
					UsePassphrase: false,
					AESMode:       "cbc",
				}
				return vaultPath, params
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *config.KeyConfig, vaultPath string) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned, got nil")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.IV == "" {
					t.Error("Expected IV to be generated for CBC mode")
				}
				if result.AESConfig.Nonce != "" {
					t.Error("Expected no nonce for CBC mode")
				}
			},
		},
		{
			name: "generate AES key with PBKDF2",
			setupFunc: func(t *testing.T) (string, KeyGenParams) {
				vaultPath := testutil.TempDir(t, "vault-pbkdf2")
				testutil.CreateTestVaultStructure(t, vaultPath)
				params := KeyGenParams{
					KeyType:          "aes",
					UsePassphrase:    true,
					AESMode:          "gcm",
					UseScrypt:        false, // Use PBKDF2
					PBKDF2Iterations: 15000,
				}
				return vaultPath, params
			},
			wantErr: false,
			validateResult: func(t *testing.T, result *config.KeyConfig, vaultPath string) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned, got nil")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.PBKDF2I != 15000 {
					t.Errorf("Expected PBKDF2 iterations to be 15000, got %d", result.AESConfig.PBKDF2I)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if short mode and this is a long-running test
			if testing.Short() && strings.Contains(tt.name, "passphrase") {
				t.Skip("Skipping passphrase test in short mode")
			}

			vaultPath, params := tt.setupFunc(t)

			// Create a mock command for testing
			cmd := &cobra.Command{}
			cmd.Flags().Bool("passphrase", params.UsePassphrase, "")
			cmd.Flags().String("passphrase-value", "testpassword123", "")
			cmd.Flags().Bool("interactive", false, "")

			result, err := HandleKeyGeneration(cmd, vaultPath, params)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, vaultPath)
			}
		})
	}
}

func TestImportKeyFromFile(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (sourceFile, destPath string)
		wantErr      bool
		errContains  string
		validateFunc func(t *testing.T, destPath string)
	}{
		{
			name: "import valid key file",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := testutil.TempDir(t, "import-test")
				sourceFile := testutil.CreateTestFile(t, tempDir, "source.key", "secret-key-data-1234567890123456")
				destPath := filepath.Join(tempDir, "dest", "secret.key")
				return sourceFile, destPath
			},
			wantErr: false,
			validateFunc: func(t *testing.T, destPath string) {
				// Check file exists
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					t.Error("Expected destination file to exist")
					return
				}

				// Check file permissions
				info, err := os.Stat(destPath)
				if err != nil {
					t.Fatalf("Failed to stat destination file: %v", err)
				}
				expectedPerms := os.FileMode(0o600)
				if info.Mode().Perm() != expectedPerms {
					t.Errorf("Expected file permissions %v, got %v", expectedPerms, info.Mode().Perm())
				}

				// Check content
				content, err := os.ReadFile(destPath)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}
				expected := "secret-key-data-1234567890123456"
				if string(content) != expected {
					t.Errorf("Expected content %q, got %q", expected, string(content))
				}
			},
		},
		{
			name: "import from non-existent file",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := testutil.TempDir(t, "import-nonexistent")
				sourceFile := filepath.Join(tempDir, "nonexistent.key")
				destPath := filepath.Join(tempDir, "dest", "secret.key")
				return sourceFile, destPath
			},
			wantErr:     true,
			errContains: "failed to read key file",
		},
		{
			name: "import to invalid destination",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := testutil.TempDir(t, "import-invalid-dest")
				sourceFile := testutil.CreateTestFile(t, tempDir, "source.key", "test-key-data")

				// Try to write to a directory that can't be created (permission denied)
				destPath := "/root/impossible/secret.key"
				return sourceFile, destPath
			},
			wantErr:     true,
			errContains: "failed to create key directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile, destPath := tt.setupFunc(t)

			err := importKeyFromFile(sourceFile, destPath)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, destPath)
			}
		})
	}
}

func TestGenerateNewKey(t *testing.T) {
	tests := []struct {
		name         string
		params       KeyGenParams
		setupCmd     func() *cobra.Command
		wantErr      bool
		errContains  string
		validateFunc func(t *testing.T, result *config.KeyConfig)
	}{
		{
			name: "generate AES key with all parameters",
			params: KeyGenParams{
				KeyType:          "aes",
				UsePassphrase:    true,
				AESMode:          "gcm",
				UseScrypt:        true,
				ScryptN:          16384,
				ScryptR:          8,
				ScryptP:          1,
				PBKDF2Iterations: 10000,
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("passphrase", true, "")
				cmd.Flags().String("passphrase-value", "testpassword123", "")
				cmd.Flags().Bool("interactive", false, "")
				return cmd
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result *config.KeyConfig) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.KDF != "scrypt" {
					t.Errorf("Expected KDF to be scrypt, got %s", result.AESConfig.KDF)
				}
				if result.AESConfig.ScryptN != 16384 {
					t.Errorf("Expected ScryptN to be 16384, got %d", result.AESConfig.ScryptN)
				}
				if result.AESConfig.Key == "" {
					t.Error("Expected key to be generated")
				}

				// Validate that the key is properly base64 encoded
				_, err := base64.StdEncoding.DecodeString(result.AESConfig.Key)
				if err != nil {
					t.Errorf("Expected key to be valid base64, got decode error: %v", err)
				}
			},
		},
		{
			name: "generate AES key without passphrase",
			params: KeyGenParams{
				KeyType:       "aes",
				UsePassphrase: false,
				AESMode:       "cbc",
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("passphrase", false, "")
				cmd.Flags().Bool("interactive", false, "")
				return cmd
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result *config.KeyConfig) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned")
				}
				if result.AESConfig == nil {
					t.Fatal("Expected AESConfig to be present")
				}
				if result.AESConfig.IV == "" {
					t.Error("Expected IV to be generated for CBC mode")
				}
				if result.Salt != "" {
					t.Error("Expected no salt for non-passphrase key")
				}
			},
		},
		{
			name: "unsupported key type",
			params: KeyGenParams{
				KeyType:       "unsupported",
				UsePassphrase: false,
			},
			setupCmd: func() *cobra.Command {
				cmd := &cobra.Command{}
				cmd.Flags().Bool("passphrase", false, "")
				cmd.Flags().Bool("interactive", false, "")
				return cmd
			},
			wantErr:     true, // Now correctly returns error for unsupported types
			errContains: "unsupported encryption type",
			validateFunc: func(t *testing.T, result *config.KeyConfig) {
				if result == nil {
					t.Fatal("Expected KeyConfig to be returned")
				}
				// The key type is stored in the vault config, not the key config
				// AES key generation still works regardless of the type field
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "generate-key-test")
			keyPath := filepath.Join(tempDir, ".sietch", "keys", "secret.key")

			// Ensure the directory exists
			if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
				t.Fatalf("Failed to create key directory: %v", err)
			}

			cmd := tt.setupCmd()
			result, err := generateNewKey(cmd, keyPath, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestKeyGenParams(t *testing.T) {
	// Test that KeyGenParams struct works as expected
	params := KeyGenParams{
		KeyType:          "aes",
		UsePassphrase:    true,
		KeyFile:          "/path/to/key",
		AESMode:          "gcm",
		UseScrypt:        true,
		ScryptN:          32768,
		ScryptR:          8,
		ScryptP:          1,
		PBKDF2Iterations: 10000,
	}

	// Verify all fields are set correctly
	if params.KeyType != "aes" {
		t.Errorf("Expected KeyType to be 'aes', got %s", params.KeyType)
	}
	if !params.UsePassphrase {
		t.Error("Expected UsePassphrase to be true")
	}
	if params.KeyFile != "/path/to/key" {
		t.Errorf("Expected KeyFile to be '/path/to/key', got %s", params.KeyFile)
	}
	if params.AESMode != "gcm" {
		t.Errorf("Expected AESMode to be 'gcm', got %s", params.AESMode)
	}
	if !params.UseScrypt {
		t.Error("Expected UseScrypt to be true")
	}
	if params.ScryptN != 32768 {
		t.Errorf("Expected ScryptN to be 32768, got %d", params.ScryptN)
	}
	if params.ScryptR != 8 {
		t.Errorf("Expected ScryptR to be 8, got %d", params.ScryptR)
	}
	if params.ScryptP != 1 {
		t.Errorf("Expected ScryptP to be 1, got %d", params.ScryptP)
	}
	if params.PBKDF2Iterations != 10000 {
		t.Errorf("Expected PBKDF2Iterations to be 10000, got %d", params.PBKDF2Iterations)
	}
}

func TestHandleKeyGenerationEdgeCases(t *testing.T) {
	t.Run("nil command", func(t *testing.T) {
		vaultPath := testutil.TempDir(t, "nil-cmd-test")
		testutil.CreateTestVaultStructure(t, vaultPath)

		params := KeyGenParams{
			KeyType:       "aes",
			UsePassphrase: false,
		}

		result, err := HandleKeyGeneration(nil, vaultPath, params)
		if err != nil {
			t.Errorf("Expected no error with nil command, got: %v", err)
		}
		if result == nil {
			t.Error("Expected result to be returned even with nil command")
		}
	})

	t.Run("empty vault path", func(t *testing.T) {
		cmd := &cobra.Command{}
		params := KeyGenParams{
			KeyType:       "aes",
			UsePassphrase: false,
		}

		// Empty vault path still creates a key path as ".sietch/keys/secret.key"
		// The function doesn't validate the vault path itself
		cmd.Flags().Bool("passphrase", false, "")
		cmd.Flags().Bool("interactive", false, "")

		result, err := HandleKeyGeneration(cmd, "", params)
		if err != nil {
			t.Errorf("Unexpected error with empty vault path: %v", err)
		}
		if result == nil {
			t.Error("Expected result even with empty vault path")
		}
	})

	t.Run("both key file and generation", func(t *testing.T) {
		vaultPath := testutil.TempDir(t, "both-key-test")
		testutil.CreateTestVaultStructure(t, vaultPath)

		keyFile := testutil.CreateTestFile(t, vaultPath, "test.key", "1234567890123456")

		params := KeyGenParams{
			KeyType: "aes",
			KeyFile: keyFile,
			AESMode: "gcm",
		}

		cmd := &cobra.Command{}
		result, err := HandleKeyGeneration(cmd, vaultPath, params)
		if err != nil {
			t.Errorf("Expected no error when key file is provided, got: %v", err)
		}

		// When key file is provided, it should be imported, not generated
		if result != nil {
			t.Error("Expected nil result when importing key from file")
		}

		// Verify the key was imported
		keyPath := filepath.Join(vaultPath, ".sietch", "keys", "secret.key")
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			t.Error("Expected key file to be imported")
		}
	})
}

func TestHandleKeyGenerationConsistency(t *testing.T) {
	// Test that multiple runs with same parameters produce different keys (due to randomness)
	// but same configuration structure
	vaultPath1 := testutil.TempDir(t, "consistency-test-1")
	vaultPath2 := testutil.TempDir(t, "consistency-test-2")
	testutil.CreateTestVaultStructure(t, vaultPath1)
	testutil.CreateTestVaultStructure(t, vaultPath2)

	params := KeyGenParams{
		KeyType:       "aes",
		UsePassphrase: false,
		AESMode:       "gcm",
	}

	cmd := &cobra.Command{}
	cmd.Flags().Bool("passphrase", false, "")
	cmd.Flags().Bool("interactive", false, "")

	result1, err1 := HandleKeyGeneration(cmd, vaultPath1, params)
	result2, err2 := HandleKeyGeneration(cmd, vaultPath2, params)

	if err1 != nil || err2 != nil {
		t.Fatalf("Unexpected errors: %v, %v", err1, err2)
	}

	if result1 == nil || result2 == nil {
		t.Fatal("Expected both results to be non-nil")
	}

	// Keys should be different (due to randomness)
	if result1.AESConfig.Key == result2.AESConfig.Key {
		t.Error("Expected different keys to be generated")
	}

	// But structure should be the same
	if result1.AESConfig.Mode != result2.AESConfig.Mode {
		t.Error("Expected same AES mode in both results")
	}
}

// Benchmark tests
func BenchmarkHandleKeyGeneration(b *testing.B) {
	// Create temp directory manually for benchmark
	vaultPath, err := os.MkdirTemp("", "benchmark-vault")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	b.Cleanup(func() {
		os.RemoveAll(vaultPath)
	})

	// Create vault structure manually
	dirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/sync",
		".sietch/chunks",
		"data",
	}
	for _, dir := range dirs {
		dirPath := filepath.Join(vaultPath, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			b.Fatalf("Failed to create vault directory %s: %v", dirPath, err)
		}
	}

	params := KeyGenParams{
		KeyType:       "aes",
		UsePassphrase: false,
		AESMode:       "gcm",
	}

	cmd := &cobra.Command{}
	cmd.Flags().Bool("passphrase", false, "")
	cmd.Flags().Bool("interactive", false, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use different vault paths to avoid conflicts
		testVaultPath := filepath.Join(vaultPath, "test", fmt.Sprintf("%d", i))
		os.MkdirAll(filepath.Join(testVaultPath, ".sietch", "keys"), 0o755)

		_, err := HandleKeyGeneration(cmd, testVaultPath, params)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkImportKeyFromFile(b *testing.B) {
	// Create temp directory manually for benchmark
	tempDir, err := os.MkdirTemp("", "benchmark-import")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	b.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Create source key file manually
	sourceFile := filepath.Join(tempDir, "source.key")
	if err := os.WriteFile(sourceFile, []byte("benchmark-key-data-1234567890123456"), 0o644); err != nil {
		b.Fatalf("Failed to create source key file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destPath := filepath.Join(tempDir, "dest", fmt.Sprintf("%d", i), "secret.key")
		err := importKeyFromFile(sourceFile, destPath)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkGenerateNewKey(b *testing.B) {
	// Create temp directory manually for benchmark
	tempDir, err := os.MkdirTemp("", "benchmark-generate")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	b.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	params := KeyGenParams{
		KeyType:       "aes",
		UsePassphrase: false,
		AESMode:       "gcm",
	}

	cmd := &cobra.Command{}
	cmd.Flags().Bool("passphrase", false, "")
	cmd.Flags().Bool("interactive", false, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyPath := filepath.Join(tempDir, "keys", fmt.Sprintf("%d", i), "secret.key")
		os.MkdirAll(filepath.Dir(keyPath), 0o755)

		_, err := generateNewKey(cmd, keyPath, params)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// Test helpers
func TestHelper(t *testing.T) {
	t.Helper()
	// This is a helper function to test the helper pattern used in other tests
}

// Integration test that uses actual file system operations
func TestHandleKeyGenerationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("full workflow with real vault structure", func(t *testing.T) {
		vaultPath := testutil.TempDir(t, "integration-vault")

		// Create full vault structure
		testutil.CreateTestVaultStructure(t, vaultPath)

		params := KeyGenParams{
			KeyType:          "aes",
			UsePassphrase:    true,
			AESMode:          "gcm",
			UseScrypt:        true,
			ScryptN:          32768,
			ScryptR:          8,
			ScryptP:          1,
			PBKDF2Iterations: 10000,
		}

		cmd := &cobra.Command{}
		cmd.Flags().Bool("passphrase", true, "")
		cmd.Flags().String("passphrase-value", "integration-test-pass-123", "")
		cmd.Flags().Bool("interactive", false, "")

		result, err := HandleKeyGeneration(cmd, vaultPath, params)
		if err != nil {
			t.Fatalf("Integration test failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result from integration test")
		}

		// Verify all expected files and directories exist
		expectedPaths := []string{
			filepath.Join(vaultPath, ".sietch"),
			filepath.Join(vaultPath, ".sietch", "keys"),
		}

		for _, expectedPath := range expectedPaths {
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Errorf("Expected path %s to exist", expectedPath)
			}
		}

		// Verify the key configuration is complete
		if result.AESConfig.Key == "" {
			t.Error("Expected key to be present in result")
		}
		if result.AESConfig.KDF != "scrypt" {
			t.Error("Expected KDF to be scrypt")
		}
		if result.Salt == "" {
			t.Error("Expected salt to be present for passphrase-protected key")
		}
	})
}
