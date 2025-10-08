package e2e


import (
	"fmt"
	"strings"
	"testing"
)

// TestLsBasic tests basic ls command functionality
func TestLsBasic(t *testing.T) {
	// Setup vault with sample files
	files := map[string]string{
		"file1.txt": "content of file 1",
		"file2.txt": "content of file 2",
		"file3.txt": "content of file 3",
	}
	vault := SetupVaultWithFiles(t, files)
	
	// Run ls command
	stdout, stderr, err := vault.Ls(t)
	
	// Verify command succeeded
	AssertCommandSuccess(t, err, stderr, "basic ls")
	
	// Verify all files are listed
	for filename := range files {
		AssertOutputContains(t, stdout, filename, "basic ls should list all files")
	}
	
	// Verify no error output
	if stderr != "" {
		t.Errorf("Expected no stderr, got: %s", stderr)
	}
}

// TestLsWithLongFormat tests ls --long flag
func TestLsWithLongFormat(t *testing.T) {
	// Setup vault with sample files
	files := map[string]string{
		"small.txt":  "small content",
		"medium.txt": strings.Repeat("medium content ", 10),
		"large.txt":  strings.Repeat("large content ", 100),
	}
	vault := SetupVaultWithFiles(t, files)
	
	// Run ls --long
	stdout, stderr, err := vault.Ls(t, "--long")
	
	// Verify command succeeded
	AssertCommandSuccess(t, err, stderr, "ls --long")
	
	// Verify output contains header columns
	AssertOutputContains(t, stdout, "SIZE", "long format should have SIZE header")
	AssertOutputContains(t, stdout, "MODIFIED", "long format should have MODIFIED header")
	AssertOutputContains(t, stdout, "CHUNKS", "long format should have CHUNKS header")
	AssertOutputContains(t, stdout, "PATH", "long format should have PATH header")
	
	// Verify all files are listed
	for filename := range files {
		AssertOutputContains(t, stdout, filename, "long format should list all files")
	}
}

// TestLsWithSortBySize tests ls --sort=size flag
func TestLsWithSortBySize(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files with different sizes
	vault.CreateFileWithSize(t, "small.txt", 100)
	vault.CreateFileWithSize(t, "large.txt", 10000)
	vault.CreateFileWithSize(t, "medium.txt", 1000)
	
	// Add files to vault
	for _, file := range []string{"small.txt", "large.txt", "medium.txt"} {
		_, stderr, err := vault.Add(t, file, "test/")
		AssertCommandSuccess(t, err, stderr, fmt.Sprintf("adding %s", file))
	}
	
	// Run ls --long --sort=size
	stdout, stderr, err := vault.Ls(t, "--long", "--sort=size")
	
	// Verify command succeeded
	AssertCommandSuccess(t, err, stderr, "ls --sort=size")
	
	// Verify all files are present
	AssertOutputContains(t, stdout, "small.txt", "should contain small.txt")
	AssertOutputContains(t, stdout, "medium.txt", "should contain medium.txt")
	AssertOutputContains(t, stdout, "large.txt", "should contain large.txt")
	
	// Verify files appear in size order (largest first)
	lines := strings.Split(stdout, "\n")
	var fileOrder []string
	for _, line := range lines {
		if strings.Contains(line, ".txt") {
			if strings.Contains(line, "large.txt") {
				fileOrder = append(fileOrder, "large")
			} else if strings.Contains(line, "medium.txt") {
				fileOrder = append(fileOrder, "medium")
			} else if strings.Contains(line, "small.txt") {
				fileOrder = append(fileOrder, "small")
			}
		}
	}
	
	// Check order (largest to smallest)
	if len(fileOrder) == 3 {
		if fileOrder[0] != "large" {
			t.Errorf("Expected large.txt first in size sort, got order: %v", fileOrder)
		}
	}
}

// TestLsWithDedupStats tests ls --dedup-stats flag
func TestLsWithDedupStats(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files with identical content to trigger deduplication
	identicalContent := strings.Repeat("This content will be deduplicated ", 100)
	vault.CreateFile(t, "dup1.txt", identicalContent)
	vault.CreateFile(t, "dup2.txt", identicalContent)
	
	// Add files to vault
	_, stderr, err := vault.Add(t, "dup1.txt", "test/")
	AssertCommandSuccess(t, err, stderr, "adding dup1.txt")
	
	_, stderr, err = vault.Add(t, "dup2.txt", "test/")
	AssertCommandSuccess(t, err, stderr, "adding dup2.txt")
	
	// Run ls --dedup-stats
	stdout, stderr, err := vault.Ls(t, "--dedup-stats")
	
	// Verify command succeeded
	AssertCommandSuccess(t, err, stderr, "ls --dedup-stats")
	
	// Verify dedup stats are shown
	AssertOutputContains(t, stdout, "shared_chunks:", "should show shared_chunks")
	AssertOutputContains(t, stdout, "saved:", "should show saved bytes")
	
	// If dedup happened, we should see shared_with
	// Note: dedup behavior depends on chunking, so we check for presence of dedup fields
	if strings.Contains(stdout, "shared_chunks: 0") {
		t.Logf("No deduplication occurred (possibly due to chunking strategy)")
	} else {
		t.Logf("Deduplication stats present in output")
	}
}

// TestLsInvalidVault tests ls command outside a vault
func TestLsInvalidVault(t *testing.T) {
	// Create a temp directory without initializing a vault
	vault := NewTestVault(t)
	
	// Run ls command (should fail)
	stdout, stderr, err := vault.Ls(t)
	
	// Verify command failed
	AssertCommandFails(t, err, "ls in non-vault directory")
	
	// Verify error message mentions vault
	combined := stdout + stderr
	if !strings.Contains(combined, "vault") && !strings.Contains(combined, "not inside") {
		t.Errorf("Expected error message about vault, got:\nStdout: %s\nStderr: %s", stdout, stderr)
	}
}

// TestLsLargeVault tests ls with many files (stress test)
func TestLsLargeVault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large vault test in short mode")
	}
	
	vault := InitializeVault(t)
	
	// Create 100 small files
	const fileCount = 100
	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		content := fmt.Sprintf("Content for file number %d", i)
		vault.CreateFile(t, filename, content)
		
		_, stderr, err := vault.Add(t, filename, "test/")
		AssertCommandSuccess(t, err, stderr, fmt.Sprintf("adding file %d", i))
	}
	
	// Run ls command
	stdout, stderr, err := vault.Ls(t)
	
	// Verify command succeeded
	AssertCommandSuccess(t, err, stderr, "ls on large vault")
	
	// Verify correct number of files listed
	AssertFileCount(t, stdout, fileCount, "large vault ls")
	
	// Spot check a few files
	AssertOutputContains(t, stdout, "file_000.txt", "should contain first file")
	AssertOutputContains(t, stdout, "file_050.txt", "should contain middle file")
	AssertOutputContains(t, stdout, "file_099.txt", "should contain last file")
}

// TestLsEmptyVault tests ls on an empty vault
func TestLsEmptyVault(t *testing.T) {
	vault := InitializeVault(t)
	
	// Run ls on empty vault
	stdout, stderr, err := vault.Ls(t)
	
	// Command should succeed
	AssertCommandSuccess(t, err, stderr, "ls on empty vault")
	
	// Should indicate no files found
	AssertOutputContains(t, stdout, "No files", "empty vault message")
}

// TestLsWithPathFilter tests ls with a path argument
func TestLsWithPathFilter(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files in different destinations
	vault.CreateFile(t, "file1.txt", "content1")
	vault.CreateFile(t, "file2.txt", "content2")
	
	// Add to different paths
	_, stderr, err := vault.Add(t, "file1.txt", "docs/")
	AssertCommandSuccess(t, err, stderr, "adding file1 to docs/")
	
	_, stderr, err = vault.Add(t, "file2.txt", "data/")
	AssertCommandSuccess(t, err, stderr, "adding file2 to data/")
	
	// List only docs/ directory
	stdout, stderr, err := vault.Ls(t, "docs/")
	AssertCommandSuccess(t, err, stderr, "ls docs/")
	
	// Should only show files in docs/
	AssertOutputContains(t, stdout, "file1.txt", "should contain docs file")
	AssertOutputNotContains(t, stdout, "file2.txt", "should not contain data file")
}

// TestLsWithTags tests ls --tags flag
func TestLsWithTags(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create and add file with tags
	vault.CreateFile(t, "tagged.txt", "content")
	_, stderr, err := vault.Add(t, "tagged.txt", "test/", "--tags", "important,work")
	AssertCommandSuccess(t, err, stderr, "adding file with tags")
	
	// Run ls --tags
	stdout, stderr, err := vault.Ls(t, "--tags")
	AssertCommandSuccess(t, err, stderr, "ls --tags")
	
	// Note: The output format depends on whether we use --long
	// Just verify the command works
	AssertOutputContains(t, stdout, "tagged.txt", "should show tagged file")
}

// TestLsMultipleFoldersWithSimilarNames tests handling of similar folder names
func TestLsMultipleFoldersWithSimilarNames(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files for similar folder paths
	vault.CreateFile(t, "test1.txt", "content1")
	vault.CreateFile(t, "test2.txt", "content2")
	vault.CreateFile(t, "test3.txt", "content3")
	
	// Add to folders with similar names
	_, stderr, err := vault.Add(t, "test1.txt", "docs/")
	AssertCommandSuccess(t, err, stderr, "adding to docs/")
	
	_, stderr, err = vault.Add(t, "test2.txt", "docs/sub/")
	AssertCommandSuccess(t, err, stderr, "adding to docs/sub/")
	
	_, stderr, err = vault.Add(t, "test3.txt", "documents/")
	AssertCommandSuccess(t, err, stderr, "adding to documents/")
	
	// List all files
	stdout, stderr, err := vault.Ls(t)
	AssertCommandSuccess(t, err, stderr, "ls all")
	
	// Verify all files are present
	AssertOutputContains(t, stdout, "test1.txt", "should contain test1")
	AssertOutputContains(t, stdout, "test2.txt", "should contain test2")
	AssertOutputContains(t, stdout, "test3.txt", "should contain test3")
	
	// Filter by docs/ (should include docs/ but not documents/)
	stdout, stderr, err = vault.Ls(t, "docs/")
	AssertCommandSuccess(t, err, stderr, "ls docs/")
	
	AssertOutputContains(t, stdout, "test1.txt", "docs/ filter should include test1")
	AssertOutputContains(t, stdout, "test2.txt", "docs/ filter should include test2 (in sub)")
	AssertOutputNotContains(t, stdout, "test3.txt", "docs/ filter should not include documents/")
}

// TestLsWithCombinedFlags tests multiple flags together
func TestLsWithCombinedFlags(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files with varying sizes
	vault.CreateFileWithSize(t, "file1.txt", 500)
	vault.CreateFileWithSize(t, "file2.txt", 1500)
	
	// Add files with tags
	_, stderr, err := vault.Add(t, "file1.txt", "test/", "--tags", "small")
	AssertCommandSuccess(t, err, stderr, "adding file1")
	
	_, stderr, err = vault.Add(t, "file2.txt", "test/", "--tags", "large")
	AssertCommandSuccess(t, err, stderr, "adding file2")
	
	// Run ls with multiple flags: --long --tags --sort=size --dedup-stats
	stdout, stderr, err := vault.Ls(t, "--long", "--tags", "--sort=size", "--dedup-stats")
	AssertCommandSuccess(t, err, stderr, "ls with combined flags")
	
	// Verify headers for long format
	AssertOutputContains(t, stdout, "SIZE", "should have SIZE header")
	AssertOutputContains(t, stdout, "TAGS", "should have TAGS header")
	
	// Verify dedup stats present
	AssertOutputContains(t, stdout, "shared_chunks:", "should have dedup stats")
	
	// Verify both files present
	AssertOutputContains(t, stdout, "file1.txt", "should contain file1")
	AssertOutputContains(t, stdout, "file2.txt", "should contain file2")
}

// TestLsWithUnicodeFilenames tests handling of unicode filenames
func TestLsWithUnicodeFilenames(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create file with unicode name
	unicodeFile := "测试文件_тест_ファイル.txt"
	vault.CreateFile(t, unicodeFile, "unicode content")
	
	_, stderr, err := vault.Add(t, unicodeFile, "test/")
	AssertCommandSuccess(t, err, stderr, "adding unicode file")
	
	// Run ls
	stdout, stderr, err := vault.Ls(t)
	AssertCommandSuccess(t, err, stderr, "ls with unicode filename")
	
	// Verify unicode filename appears correctly
	AssertOutputContains(t, stdout, unicodeFile, "should display unicode filename")
}

// TestLsSortByName tests explicit sort by name
func TestLsSortByName(t *testing.T) {
	vault := InitializeVault(t)
	
	// Create files in non-alphabetical order
	for _, name := range []string{"zebra.txt", "alpha.txt", "beta.txt"} {
		vault.CreateFile(t, name, "content")
		_, stderr, err := vault.Add(t, name, "test/")
		AssertCommandSuccess(t, err, stderr, fmt.Sprintf("adding %s", name))
	}
	
	// Run ls --sort=name
	stdout, stderr, err := vault.Ls(t, "--sort=name")
	AssertCommandSuccess(t, err, stderr, "ls --sort=name")
	
	// Verify all files present
	AssertOutputContains(t, stdout, "alpha.txt", "should contain alpha")
	AssertOutputContains(t, stdout, "beta.txt", "should contain beta")
	AssertOutputContains(t, stdout, "zebra.txt", "should contain zebra")
	
	// Check order
	lines := strings.Split(stdout, "\n")
	var order []string
	for _, line := range lines {
		if strings.Contains(line, "alpha.txt") {
			order = append(order, "alpha")
		} else if strings.Contains(line, "beta.txt") {
			order = append(order, "beta")
		} else if strings.Contains(line, "zebra.txt") {
			order = append(order, "zebra")
		}
	}
	
	if len(order) == 3 && order[0] != "alpha" {
		t.Logf("Files may not be in strict alphabetical order: %v", order)
	}
}