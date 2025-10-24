package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
)

// TestLargeScaleDeduplication tests deduplication with large numbers of files and complex scenarios
func TestLargeScaleDeduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large-scale tests in short mode")
	}

	t.Run("DeduplicationWith100Files", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "large-scale-100-files")

		// Create 100 test files with some duplicated content
		filesDir := filepath.Join(vaultPath, "files")
		err := os.MkdirAll(filesDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create files directory: %v", err)
		}

		// Create file manifests to simulate chunked files
		manifestsDir := filepath.Join(vaultPath, ".sietch", "manifests")
		err = os.MkdirAll(manifestsDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create manifests directory: %v", err)
		}

		// Create chunks directory
		chunksDir := filepath.Join(vaultPath, ".sietch", "chunks")
		err = os.MkdirAll(chunksDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create chunks directory: %v", err)
		}

		// Create deduplication manager for testing
		dedupConfig := config.DeduplicationConfig{
			Enabled:      true,
			Strategy:     "content",
			MinChunkSize: "1KB",
			MaxChunkSize: "64MB",
			GCThreshold:  1000,
			IndexEnabled: true,
		}

		manager, err := deduplication.NewManager(vaultPath, dedupConfig)
		if err != nil {
			t.Fatalf("Failed to create deduplication manager: %v", err)
		}

		// Create 100 files with varying content patterns
		numFiles := 100
		duplicatePatterns := []string{
			"common content pattern 1",
			"common content pattern 2",
			"common content pattern 3",
		}

		for i := 0; i < numFiles; i++ {
			fileName := fmt.Sprintf("file_%03d.txt", i)
			filePath := filepath.Join(filesDir, fileName)

			var content string
			if i%10 == 0 {
				// Every 10th file uses a duplicate pattern
				content = duplicatePatterns[i%len(duplicatePatterns)]
			} else {
				// Unique content
				content = fmt.Sprintf("unique content for file %d\nwith multiple lines\nand more data", i)
			}

			err := os.WriteFile(filePath, []byte(content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", fileName, err)
			}

			// Create corresponding chunk and add to deduplication index
			chunkHash := fmt.Sprintf("hash_%03d", i)
			chunkRef := config.ChunkRef{
				Hash: chunkHash,
				Size: int64(len(content)),
			}

			_, _, err = manager.ProcessChunk(chunkRef, []byte(content), chunkHash)
			if err != nil {
				t.Fatalf("Failed to process chunk for file %s: %v", fileName, err)
			}
		}

		// Save the deduplication index
		err = manager.Save()
		if err != nil {
			t.Fatalf("Failed to save deduplication index: %v", err)
		}

		// Change to vault directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test stats command with 100 files
		statsCmd := dedupStatsCmd
		statsCmd.SetArgs([]string{})

		statsOutput := captureOutput(func() {
			err := statsCmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed: %v", err)
			}
		})

		if !strings.Contains(statsOutput, "Deduplication Statistics:") {
			t.Error("Stats output should contain statistics header")
		}

		// Verify that stats show multiple chunks
		if !strings.Contains(statsOutput, "Total chunks:") {
			t.Error("Stats should show total chunks")
		}

		t.Logf("Stats output for 100 files:\n%s", statsOutput)

		// Test GC command
		gcCmd := dedupGcCmd
		gcCmd.SetArgs([]string{})

		gcOutput := captureOutput(func() {
			err := gcCmd.Execute()
			if err != nil {
				t.Errorf("GC command failed: %v", err)
			}
		})

		if !strings.Contains(gcOutput, "Garbage collection completed") {
			t.Error("GC should complete successfully")
		}

		t.Logf("GC output for 100 files:\n%s", gcOutput)

		// Test optimize command
		optimizeCmd := dedupOptimizeCmd
		optimizeCmd.SetArgs([]string{})

		optimizeOutput := captureOutput(func() {
			err := optimizeCmd.Execute()
			if err != nil {
				t.Errorf("Optimize command failed: %v", err)
			}
		})

		if !strings.Contains(optimizeOutput, "Optimization Results:") {
			t.Error("Optimize should show results")
		}

		t.Logf("Optimize output for 100 files:\n%s", optimizeOutput)
	})

	t.Run("MultipleDirectoriesWithSimilarNames", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "similar-names")

		// Create multiple directories with similar names
		dirNames := []string{
			"documents",
			"Documents",
			"document",
			"docs",
			"Docs",
			"documentation",
			"images",
			"Images",
			"image",
			"img",
		}

		for _, dirName := range dirNames {
			dirPath := filepath.Join(vaultPath, dirName)
			err := os.MkdirAll(dirPath, 0o755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dirName, err)
			}

			// Create files in each directory
			for i := 0; i < 5; i++ {
				fileName := fmt.Sprintf("file%d.txt", i)
				filePath := filepath.Join(dirPath, fileName)
				content := fmt.Sprintf("Content for %s in %s", fileName, dirName)

				err := os.WriteFile(filePath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to create file %s in %s: %v", fileName, dirName, err)
				}
			}
		}

		// Create some dummy chunks
		chunksDir := filepath.Join(vaultPath, ".sietch", "chunks")
		err := os.MkdirAll(chunksDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create chunks directory: %v", err)
		}

		for i := 0; i < 20; i++ {
			chunkFile := filepath.Join(chunksDir, fmt.Sprintf("chunk_%03d.dat", i))
			err := os.WriteFile(chunkFile, []byte(fmt.Sprintf("chunk data %d", i)), 0o644)
			if err != nil {
				t.Fatalf("Failed to create chunk file: %v", err)
			}
		}

		// Change to vault directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test all dedup commands with complex directory structure
		commands := []struct {
			name string
			cmd  *cobra.Command
		}{
			{"stats", dedupStatsCmd},
			{"gc", dedupGcCmd},
			{"optimize", dedupOptimizeCmd},
		}

		for _, command := range commands {
			t.Run(fmt.Sprintf("%s_with_similar_dirs", command.name), func(t *testing.T) {
				cmd := command.cmd
				cmd.SetArgs([]string{})

				output := captureOutput(func() {
					err := cmd.Execute()
					if err != nil {
						t.Errorf("%s command failed: %v", command.name, err)
					}
				})

				t.Logf("%s output with similar directory names:\n%s", command.name, output)

				// Basic validation that command produces output
				if len(output) == 0 {
					t.Errorf("%s command should produce output", command.name)
				}
			})
		}
	})

	t.Run("PerformanceWithLargeChunks", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "large-chunks")

		// Create chunks directory
		chunksDir := filepath.Join(vaultPath, ".sietch", "chunks")
		err := os.MkdirAll(chunksDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create chunks directory: %v", err)
		}

		// Create some large chunk files (1MB each)
		largeChunkSize := 1024 * 1024 // 1MB
		numLargeChunks := 10

		for i := 0; i < numLargeChunks; i++ {
			chunkFile := filepath.Join(chunksDir, fmt.Sprintf("large_chunk_%03d.dat", i))

			// Create large chunk data
			data := make([]byte, largeChunkSize)
			for j := range data {
				data[j] = byte(j % 256)
			}

			err := os.WriteFile(chunkFile, data, 0o644)
			if err != nil {
				t.Fatalf("Failed to create large chunk file: %v", err)
			}
		}

		// Change to vault directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test performance with large chunks
		statsCmd := dedupStatsCmd
		statsCmd.SetArgs([]string{})

		statsOutput := captureOutput(func() {
			err := statsCmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed with large chunks: %v", err)
			}
		})

		if !strings.Contains(statsOutput, "Deduplication Statistics:") {
			t.Error("Stats should work with large chunks")
		}

		t.Logf("Performance test output with large chunks:\n%s", statsOutput)
	})

	t.Run("ConcurrentOperationsSimulation", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "concurrent-ops")

		// Create deduplication manager
		dedupConfig := config.DeduplicationConfig{
			Enabled:      true,
			Strategy:     "content",
			MinChunkSize: "1KB",
			MaxChunkSize: "64MB",
			GCThreshold:  100,
			IndexEnabled: true,
		}

		manager, err := deduplication.NewManager(vaultPath, dedupConfig)
		if err != nil {
			t.Fatalf("Failed to create deduplication manager: %v", err)
		}

		// Simulate concurrent operations by adding many chunks rapidly
		numChunks := 50
		for i := 0; i < numChunks; i++ {
			chunkData := fmt.Sprintf("chunk data %d with some additional content to make it larger", i)
			chunkRef := config.ChunkRef{
				Hash: fmt.Sprintf("concurrent_chunk_%d", i),
				Size: int64(len(chunkData)),
			}

			_, _, err := manager.ProcessChunk(chunkRef, []byte(chunkData), fmt.Sprintf("storage_%d", i))
			if err != nil {
				t.Fatalf("Failed to process chunk %d: %v", i, err)
			}
		}

		// Save the index
		err = manager.Save()
		if err != nil {
			t.Fatalf("Failed to save index: %v", err)
		}

		// Change to vault directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test that all commands work after concurrent-like operations
		statsCmd := dedupStatsCmd
		statsCmd.SetArgs([]string{})

		statsOutput := captureOutput(func() {
			err := statsCmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed after concurrent operations: %v", err)
			}
		})

		if !strings.Contains(statsOutput, "Total chunks:") {
			t.Error("Stats should show chunk information")
		}

		// Test garbage collection
		gcCmd := dedupGcCmd
		gcCmd.SetArgs([]string{})

		gcOutput := captureOutput(func() {
			err := gcCmd.Execute()
			if err != nil {
				t.Errorf("GC command failed after concurrent operations: %v", err)
			}
		})

		if !strings.Contains(gcOutput, "Garbage collection completed") {
			t.Error("GC should complete successfully")
		}

		t.Logf("Concurrent operations test completed successfully")
	})
}

// TestEdgeCaseScenarios tests various edge cases and boundary conditions
func TestEdgeCaseScenarios(t *testing.T) {
	t.Run("EmptyVaultWithManyCommands", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "empty-many-commands")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Run each command multiple times to test stability
		for i := 0; i < 5; i++ {
			t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
				// Stats
				statsCmd := dedupStatsCmd
				statsCmd.SetArgs([]string{})
				_ = captureOutput(func() {
					err := statsCmd.Execute()
					if err != nil {
						t.Errorf("Stats command failed on iteration %d: %v", i, err)
					}
				})

				// GC
				gcCmd := dedupGcCmd
				gcCmd.SetArgs([]string{})
				_ = captureOutput(func() {
					err := gcCmd.Execute()
					if err != nil {
						t.Errorf("GC command failed on iteration %d: %v", i, err)
					}
				})

				// Optimize
				optimizeCmd := dedupOptimizeCmd
				optimizeCmd.SetArgs([]string{})
				_ = captureOutput(func() {
					err := optimizeCmd.Execute()
					if err != nil {
						t.Errorf("Optimize command failed on iteration %d: %v", i, err)
					}
				})
			})
		}
	})

	t.Run("VaultWithSpecialCharacterNames", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "special-chars")

		// Create directories and files with special characters
		specialNames := []string{
			"file with spaces.txt",
			"file-with-dashes.txt",
			"file_with_underscores.txt",
			"file.with.dots.txt",
			"file(with)parentheses.txt",
			"file[with]brackets.txt",
			"file{with}braces.txt",
		}

		for _, name := range specialNames {
			filePath := filepath.Join(vaultPath, name)
			content := fmt.Sprintf("Content for %s", name)
			err := os.WriteFile(filePath, []byte(content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create file with special characters: %v", err)
			}
		}

		// Create some chunks
		chunksDir := filepath.Join(vaultPath, ".sietch", "chunks")
		err := os.MkdirAll(chunksDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create chunks directory: %v", err)
		}

		for i, name := range specialNames {
			chunkFile := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d.dat", i))
			err := os.WriteFile(chunkFile, []byte(fmt.Sprintf("chunk for %s", name)), 0o644)
			if err != nil {
				t.Fatalf("Failed to create chunk file: %v", err)
			}
		}

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test that commands work with special character filenames
		statsCmd := dedupStatsCmd
		statsCmd.SetArgs([]string{})

		statsOutput := captureOutput(func() {
			err := statsCmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed with special characters: %v", err)
			}
		})

		if !strings.Contains(statsOutput, "Deduplication Statistics:") {
			t.Error("Stats should work with special character filenames")
		}

		t.Logf("Special characters test completed successfully")
	})

	t.Run("DeepDirectoryNesting", func(t *testing.T) {
		vaultPath := setupLargeTestVault(t, "deep-nesting")

		// Create deeply nested directory structure
		deepPath := vaultPath
		for i := 0; i < 10; i++ {
			deepPath = filepath.Join(deepPath, fmt.Sprintf("level_%d", i))
			err := os.MkdirAll(deepPath, 0o755)
			if err != nil {
				t.Fatalf("Failed to create deep directory level %d: %v", i, err)
			}

			// Create a file at each level
			fileName := fmt.Sprintf("file_at_level_%d.txt", i)
			filePath := filepath.Join(deepPath, fileName)
			content := fmt.Sprintf("Content at directory level %d", i)
			err = os.WriteFile(filePath, []byte(content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create file at level %d: %v", i, err)
			}
		}

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		// Test that commands work with deep directory nesting
		statsCmd := dedupStatsCmd
		statsCmd.SetArgs([]string{})

		statsOutput := captureOutput(func() {
			err := statsCmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed with deep nesting: %v", err)
			}
		})

		if !strings.Contains(statsOutput, "Deduplication Statistics:") {
			t.Error("Stats should work with deeply nested directories")
		}

		t.Logf("Deep directory nesting test completed successfully")
	})
}

// Helper function for large-scale tests
func setupLargeTestVault(t *testing.T, name string) string {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", name)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Create vault structure
	err = os.MkdirAll(filepath.Join(tempDir, ".sietch", "chunks"), 0o755)
	if err != nil {
		t.Fatalf("Failed to create vault structure: %v", err)
	}

	// Create a basic vault configuration with deduplication enabled
	vaultConfig := &config.VaultConfig{
		Deduplication: config.DeduplicationConfig{
			Enabled:      true,
			Strategy:     "content",
			MinChunkSize: "1KB",
			MaxChunkSize: "64MB",
			GCThreshold:  1000,
			IndexEnabled: true,
		},
	}

	err = config.SaveVaultConfig(tempDir, vaultConfig)
	if err != nil {
		t.Fatalf("Failed to save vault config: %v", err)
	}

	return tempDir
}
