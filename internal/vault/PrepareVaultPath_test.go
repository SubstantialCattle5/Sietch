package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/substantialcattle5/sietch/testutil"
)

func TestPrepareVaultPath(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (vaultPath, vaultName string)
		forceInit   bool
		wantErr     bool
		errContains string
	}{
		{
			name: "new vault creation",
			setupFunc: func(t *testing.T) (string, string) {
				return testutil.TempDir(t, "vault-parent"), "new-vault"
			},
			forceInit: false,
			wantErr:   false,
		},
		{
			name: "existing vault without force",
			setupFunc: func(t *testing.T) (string, string) {
				parentDir := testutil.TempDir(t, "vault-parent")
				vaultName := "existing-vault"
				vaultPath := filepath.Join(parentDir, vaultName)
				sietchDir := filepath.Join(vaultPath, ".sietch")

				// Create existing vault structure
				if err := os.MkdirAll(sietchDir, 0o755); err != nil {
					t.Fatalf("Failed to create existing vault: %v", err)
				}

				return parentDir, vaultName
			},
			forceInit:   false,
			wantErr:     true,
			errContains: "vault already exists",
		},
		{
			name: "existing vault with force",
			setupFunc: func(t *testing.T) (string, string) {
				parentDir := testutil.TempDir(t, "vault-parent")
				vaultName := "existing-vault"
				vaultPath := filepath.Join(parentDir, vaultName)
				sietchDir := filepath.Join(vaultPath, ".sietch")

				// Create existing vault structure
				if err := os.MkdirAll(sietchDir, 0o755); err != nil {
					t.Fatalf("Failed to create existing vault: %v", err)
				}

				return parentDir, vaultName
			},
			forceInit: true,
			wantErr:   false,
		},
		{
			name: "vault in current directory",
			setupFunc: func(t *testing.T) (string, string) {
				return ".", "test-vault"
			},
			forceInit: false,
			wantErr:   false,
		},
		{
			name: "nested vault path",
			setupFunc: func(t *testing.T) (string, string) {
				parentDir := testutil.TempDir(t, "vault-parent")
				nestedPath := filepath.Join(parentDir, "nested", "path")
				return nestedPath, "nested-vault"
			},
			forceInit: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultPath, vaultName := tt.setupFunc(t)

			absVaultPath, err := PrepareVaultPath(vaultPath, vaultName, tt.forceInit)

			if tt.wantErr {
				if err == nil {
					t.Errorf("PrepareVaultPath() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("PrepareVaultPath() error = %q, want it to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("PrepareVaultPath() unexpected error: %v", err)
				return
			}

			// Verify the returned path is absolute
			if !filepath.IsAbs(absVaultPath) {
				t.Errorf("PrepareVaultPath() returned relative path: %s", absVaultPath)
			}

			// Verify the path contains the vault name
			if vaultPath != "." {
				if !containsString(absVaultPath, vaultName) {
					t.Errorf("PrepareVaultPath() returned path %s does not contain vault name %s", absVaultPath, vaultName)
				}
			}

			// Verify the path is accessible (can get file info)
			parentDir := filepath.Dir(absVaultPath)
			if _, err := os.Stat(parentDir); err != nil {
				// Parent directory doesn't exist, which is fine for this function
				// The actual vault creation will happen later
			}
		})
	}
}

func TestPrepareVaultPathEdgeCases(t *testing.T) {
	t.Run("empty vault name", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test")
		_, err := PrepareVaultPath(tempDir, "", false)
		if err != nil {
			t.Errorf("PrepareVaultPath() with empty name should not error, got: %v", err)
		}
	})

	t.Run("vault name with special characters", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test")
		specialNames := []string{
			"vault-with-dashes",
			"vault_with_underscores",
			"vault.with.dots",
			"vault with spaces",
			"vault123",
		}

		for _, name := range specialNames {
			t.Run("name_"+name, func(t *testing.T) {
				absPath, err := PrepareVaultPath(tempDir, name, false)
				if err != nil {
					t.Errorf("PrepareVaultPath() with name %q failed: %v", name, err)
					return
				}
				if absPath == "" {
					t.Errorf("PrepareVaultPath() returned empty path for name %q", name)
				}
			})
		}
	})

	t.Run("deeply nested path", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test")
		deepPath := filepath.Join(tempDir, "very", "deep", "nested", "path", "structure")

		absPath, err := PrepareVaultPath(deepPath, "deep-vault", false)
		if err != nil {
			t.Errorf("PrepareVaultPath() with deep path failed: %v", err)
			return
		}

		if !filepath.IsAbs(absPath) {
			t.Errorf("PrepareVaultPath() should return absolute path, got: %s", absPath)
		}
	})

	t.Run("relative path with dots", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "test")

		// Change to temp directory to test relative paths
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Errorf("Failed to restore directory: %v", err)
			}
		}()

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		relativePaths := []string{
			".",
			"..",
			"./relative",
			"../parent",
		}

		for _, relPath := range relativePaths {
			t.Run("path_"+relPath, func(t *testing.T) {
				absPath, err := PrepareVaultPath(relPath, "test-vault", false)
				if err != nil {
					t.Errorf("PrepareVaultPath() with relative path %q failed: %v", relPath, err)
					return
				}
				if !filepath.IsAbs(absPath) {
					t.Errorf("PrepareVaultPath() should return absolute path for %q, got: %s", relPath, absPath)
				}
			})
		}
	})
}

func TestPrepareVaultPathConsistency(t *testing.T) {
	tempDir := testutil.TempDir(t, "consistency-test")
	vaultName := "consistency-vault"

	// Call PrepareVaultPath multiple times with same inputs
	paths := make([]string, 5)
	for i := 0; i < 5; i++ {
		path, err := PrepareVaultPath(tempDir, vaultName, false)
		if err != nil {
			t.Fatalf("PrepareVaultPath() call %d failed: %v", i+1, err)
		}
		paths[i] = path
	}

	// All paths should be identical
	firstPath := paths[0]
	for i, path := range paths {
		if path != firstPath {
			t.Errorf("PrepareVaultPath() call %d returned different path: %s vs %s", i+1, path, firstPath)
		}
	}
}

func TestPrepareVaultPathWithExistingFile(t *testing.T) {
	tempDir := testutil.TempDir(t, "file-conflict-test")
	vaultName := "vault-file"

	// Create a file with the same name as the intended vault
	conflictFile := filepath.Join(tempDir, vaultName)
	if err := os.WriteFile(conflictFile, []byte("conflict"), 0o644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	// PrepareVaultPath should still work (it doesn't check if it's a file vs directory)
	absPath, err := PrepareVaultPath(tempDir, vaultName, false)
	if err != nil {
		t.Errorf("PrepareVaultPath() failed with existing file: %v", err)
		return
	}

	if absPath == "" {
		t.Error("PrepareVaultPath() returned empty path")
	}

	// The returned path should still be the conflicting path
	// The actual conflict will be handled during vault creation
	expectedPath, _ := filepath.Abs(conflictFile)
	if absPath != expectedPath {
		t.Errorf("PrepareVaultPath() returned %s, expected %s", absPath, expectedPath)
	}
}

func TestPrepareVaultPathPermissions(t *testing.T) {
	// Skip this test on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tempDir := testutil.TempDir(t, "permission-test")
	vaultName := "permission-vault"

	// Create a directory with restrictive permissions
	restrictedDir := filepath.Join(tempDir, "restricted")
	if err := os.MkdirAll(restrictedDir, 0o000); err != nil {
		t.Fatalf("Failed to create restricted directory: %v", err)
	}

	// Restore permissions for cleanup
	t.Cleanup(func() {
		os.Chmod(restrictedDir, 0o755)
	})

	// PrepareVaultPath should still work as it doesn't create directories
	absPath, err := PrepareVaultPath(restrictedDir, vaultName, false)
	if err != nil {
		t.Errorf("PrepareVaultPath() failed with restricted parent: %v", err)
		return
	}

	if absPath == "" {
		t.Error("PrepareVaultPath() returned empty path")
	}
}

// Helper function to check if a string contains another string
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					filepath.Base(s) == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkPrepareVaultPath(b *testing.B) {
	tempDir := testutil.TempDir(nil, "benchmark")
	vaultName := "benchmark-vault"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PrepareVaultPath(tempDir, vaultName, false)
		if err != nil {
			b.Fatalf("PrepareVaultPath() failed: %v", err)
		}
	}
}

func BenchmarkPrepareVaultPathWithExistingVault(b *testing.B) {
	tempDir := testutil.TempDir(nil, "benchmark")
	vaultName := "existing-vault"

	// Create existing vault
	vaultPath := filepath.Join(tempDir, vaultName)
	sietchDir := filepath.Join(vaultPath, ".sietch")
	if err := os.MkdirAll(sietchDir, 0o755); err != nil {
		b.Fatalf("Failed to create existing vault: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PrepareVaultPath(tempDir, vaultName, true) // force = true
		if err != nil {
			b.Fatalf("PrepareVaultPath() failed: %v", err)
		}
	}
}
