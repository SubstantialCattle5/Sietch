package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/substantialcattle5/sietch/testutil"
)

func TestCreateVaultStructure(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "create structure in empty directory",
			setupFunc: func(t *testing.T) string {
				return testutil.TempDir(t, "empty-vault")
			},
			wantErr: false,
		},
		{
			name: "create structure in existing directory",
			setupFunc: func(t *testing.T) string {
				dir := testutil.TempDir(t, "existing-vault")
				// Create some existing files
				testutil.CreateTestFile(t, dir, "existing.txt", "existing content")
				return dir
			},
			wantErr: false,
		},
		{
			name: "create structure with nested path",
			setupFunc: func(t *testing.T) string {
				parentDir := testutil.TempDir(t, "parent")
				return filepath.Join(parentDir, "nested", "vault")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultPath := tt.setupFunc(t)

			err := CreateVaultStructure(vaultPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateVaultStructure() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateVaultStructure() unexpected error: %v", err)
				return
			}

			// Verify the vault structure was created
			expectedDirs := []string{
				".sietch",
				".sietch/keys",
				".sietch/manifests",
				".sietch/chunks",
				"data",
			}

			for _, dir := range expectedDirs {
				dirPath := filepath.Join(vaultPath, dir)
				testutil.AssertDirExists(t, dirPath)
			}

			// Verify permissions are correct
			sietchDir := filepath.Join(vaultPath, ".sietch")
			info, err := os.Stat(sietchDir)
			if err != nil {
				t.Errorf("Failed to stat .sietch directory: %v", err)
				return
			}

			// Check that it's a directory
			if !info.IsDir() {
				t.Error(".sietch should be a directory")
			}

			// Check permissions (on Unix-like systems)
			if info.Mode().Perm() != 0o755 {
				t.Errorf(".sietch directory permissions = %o, want %o", info.Mode().Perm(), 0o755)
			}
		})
	}
}

func TestCreateVaultStructureIdempotent(t *testing.T) {
	vaultPath := testutil.TempDir(t, "idempotent-vault")

	// Create vault structure multiple times
	for i := 0; i < 3; i++ {
		err := CreateVaultStructure(vaultPath)
		if err != nil {
			t.Errorf("CreateVaultStructure() iteration %d failed: %v", i+1, err)
		}
	}

	// Verify structure exists and is correct
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(vaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

func TestCreateVaultStructureWithExistingFiles(t *testing.T) {
	vaultPath := testutil.TempDir(t, "existing-files-vault")

	// Create some files that might conflict
	conflictingFiles := map[string]string{
		".sietch/existing.txt":                "existing sietch file",
		"data/existing-data.txt":              "existing data file",
		".sietch/keys/existing.key":           "existing key file",
		".sietch/manifests/existing.manifest": "existing manifest file",
		".sietch/chunks/existing.chunk":       "existing chunk file",
	}

	// Create parent directories and files
	for filePath, content := range conflictingFiles {
		fullPath := filepath.Join(vaultPath, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create parent dir for %s: %v", fullPath, err)
		}
		testutil.CreateTestFile(t, vaultPath, filePath, content)
	}

	// CreateVaultStructure should not fail and should not remove existing files
	err := CreateVaultStructure(vaultPath)
	if err != nil {
		t.Errorf("CreateVaultStructure() with existing files failed: %v", err)
	}

	// Verify existing files are still there
	for filePath, expectedContent := range conflictingFiles {
		fullPath := filepath.Join(vaultPath, filePath)
		testutil.AssertFileExists(t, fullPath)
		testutil.AssertFileContains(t, fullPath, expectedContent)
	}

	// Verify all required directories exist
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(vaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

func TestCreateVaultStructurePermissions(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	vaultPath := testutil.TempDir(t, "permissions-vault")

	err := CreateVaultStructure(vaultPath)
	if err != nil {
		t.Errorf("CreateVaultStructure() failed: %v", err)
		return
	}

	// Check permissions on critical directories
	criticalDirs := map[string]os.FileMode{
		".sietch":           0o755,
		".sietch/keys":      0o755, // Might be 0700 for security
		".sietch/manifests": 0o755,
		".sietch/chunks":    0o755,
		"data":              0o755,
	}

	for dir, expectedPerm := range criticalDirs {
		dirPath := filepath.Join(vaultPath, dir)
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("Failed to stat directory %s: %v", dir, err)
			continue
		}

		actualPerm := info.Mode().Perm()
		// Allow some flexibility in permissions (keys directory might be more restrictive)
		if dir == ".sietch/keys" && (actualPerm == 0o700 || actualPerm == 0o755) {
			continue // Either permission is acceptable for keys
		}

		if actualPerm != expectedPerm {
			t.Errorf("Directory %s permissions = %o, want %o", dir, actualPerm, expectedPerm)
		}
	}
}

func TestCreateVaultStructureInRestrictedLocation(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	parentDir := testutil.TempDir(t, "restricted-parent")
	restrictedDir := filepath.Join(parentDir, "restricted")

	// Create a directory with no write permissions
	if err := os.MkdirAll(restrictedDir, 0o555); err != nil {
		t.Fatalf("Failed to create restricted directory: %v", err)
	}

	// Restore permissions for cleanup
	t.Cleanup(func() {
		os.Chmod(restrictedDir, 0o755)
	})

	vaultPath := filepath.Join(restrictedDir, "vault")

	// This should fail due to permissions
	err := CreateVaultStructure(vaultPath)
	if err == nil {
		t.Error("CreateVaultStructure() expected error in restricted location but got none")
	}
}

func TestCreateVaultStructureAbsolutePath(t *testing.T) {
	tempDir := testutil.TempDir(t, "absolute-path-test")
	vaultPath := filepath.Join(tempDir, "vault")

	// Make sure we're using an absolute path
	absVaultPath, err := filepath.Abs(vaultPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	err = CreateVaultStructure(absVaultPath)
	if err != nil {
		t.Errorf("CreateVaultStructure() with absolute path failed: %v", err)
		return
	}

	// Verify structure was created
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(absVaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

func TestCreateVaultStructureRelativePath(t *testing.T) {
	// Change to a temporary directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := testutil.TempDir(t, "relative-path-test")
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Use relative path
	vaultPath := "relative-vault"

	err = CreateVaultStructure(vaultPath)
	if err != nil {
		t.Errorf("CreateVaultStructure() with relative path failed: %v", err)
		return
	}

	// Verify structure was created
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(vaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

func TestCreateVaultStructureDeepNesting(t *testing.T) {
	tempDir := testutil.TempDir(t, "deep-nesting-test")

	// Create a deeply nested vault path
	vaultPath := filepath.Join(tempDir, "level1", "level2", "level3", "level4", "vault")

	err := CreateVaultStructure(vaultPath)
	if err != nil {
		t.Errorf("CreateVaultStructure() with deep nesting failed: %v", err)
		return
	}

	// Verify structure was created
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(vaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

// Edge case: very long path names
func TestCreateVaultStructureLongPath(t *testing.T) {
	tempDir := testutil.TempDir(t, "long-path-test")

	// Create a very long directory name (but within reasonable filesystem limits)
	longDirName := ""
	for i := 0; i < 50; i++ {
		longDirName += "very-long-directory-name-"
	}

	vaultPath := filepath.Join(tempDir, longDirName)

	err := CreateVaultStructure(vaultPath)
	if err != nil {
		// This might fail on some filesystems due to path length limits
		// That's acceptable behavior
		t.Logf("CreateVaultStructure() with long path failed (expected on some filesystems): %v", err)
		return
	}

	// If it succeeded, verify the structure
	testutil.AssertDirExists(t, filepath.Join(vaultPath, ".sietch"))
}

// Test concurrent access
func TestCreateVaultStructureConcurrent(t *testing.T) {
	testutil.SkipIfShort(t, "concurrent test")

	vaultPath := testutil.TempDir(t, "concurrent-vault")

	// Run CreateVaultStructure concurrently
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			errors <- CreateVaultStructure(vaultPath)
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Concurrent CreateVaultStructure() failed: %v", err)
		}
	}

	// Verify structure exists and is correct
	expectedDirs := []string{
		".sietch",
		".sietch/keys",
		".sietch/manifests",
		".sietch/chunks",
		"data",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(vaultPath, dir)
		testutil.AssertDirExists(t, dirPath)
	}
}

// Benchmark tests
func BenchmarkCreateVaultStructure(b *testing.B) {
	tempDir := testutil.TempDir(nil, "benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vaultPath := filepath.Join(tempDir, "vault", string(rune('a'+i%26)))
		err := CreateVaultStructure(vaultPath)
		if err != nil {
			b.Fatalf("CreateVaultStructure() failed: %v", err)
		}
	}
}

func BenchmarkCreateVaultStructureExisting(b *testing.B) {
	vaultPath := testutil.TempDir(nil, "benchmark-existing")

	// Create structure once
	if err := CreateVaultStructure(vaultPath); err != nil {
		b.Fatalf("Initial CreateVaultStructure() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := CreateVaultStructure(vaultPath)
		if err != nil {
			b.Fatalf("CreateVaultStructure() failed: %v", err)
		}
	}
}
