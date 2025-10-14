package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/testutil"
)

func TestDedupCommand(t *testing.T) {
	t.Run("DedupCommandHelp", func(t *testing.T) {
		// Create a temporary vault
		vaultPath := testutil.TempDir(t, "dedup-help-test")

		// Initialize a basic vault structure
		err := os.MkdirAll(filepath.Join(vaultPath, ".sietch"), 0o755)
		if err != nil {
			t.Fatalf("Failed to create vault structure: %v", err)
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

		// Create a root command and add dedup as subcommand for proper testing
		rootCmd := &cobra.Command{Use: "sietch"}
		rootCmd.AddCommand(dedupCmd)
		rootCmd.SetArgs([]string{"dedup", "--help"})

		// Capture output
		output := captureOutput(func() {
			_ = rootCmd.Execute() // Help command may return an error
		})

		// Debug: Print actual output
		t.Logf("Captured output: %q", output)

		// Verify help content
		if !strings.Contains(output, "Manage deduplication settings and operations") {
			t.Errorf("Help output should contain main description. Got: %q", output)
		}
		if !strings.Contains(output, "Available Commands:") {
			t.Error("Help output should contain available commands section")
		}
		if !strings.Contains(output, "stats") {
			t.Error("Help output should mention stats command")
		}
		if !strings.Contains(output, "gc") {
			t.Error("Help output should mention gc command")
		}
		if !strings.Contains(output, "optimize") {
			t.Error("Help output should mention optimize command")
		}
		if !strings.Contains(output, "--setup") {
			t.Error("Help output should mention --setup flag")
		}
	})

	t.Run("DedupCommandOutsideVault", func(t *testing.T) {
		// Test running dedup command outside a vault
		tempDir := testutil.TempDir(t, "non-vault-dir")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		cmd := dedupCmd
		cmd.SetArgs([]string{"--setup"})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when running dedup outside vault")
			}
		})

		if !strings.Contains(output, "not inside a vault") {
			t.Error("Error output should indicate not inside a vault")
		}
	})

	t.Run("DedupCommandUninitializedVault", func(t *testing.T) {
		// Create a directory with .sietch but no config
		vaultPath := testutil.TempDir(t, "uninitialized-vault")
		err := os.MkdirAll(filepath.Join(vaultPath, ".sietch"), 0o755)
		if err != nil {
			t.Fatalf("Failed to create .sietch directory: %v", err)
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

		cmd := dedupCmd
		cmd.SetArgs([]string{"--setup"})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when running dedup in uninitialized vault")
			}
		})

		if !strings.Contains(output, "vault not initialized") {
			t.Error("Error output should indicate vault not initialized")
		}
	})
}

func TestDedupStatsCommand(t *testing.T) {
	t.Run("StatsInEmptyVault", func(t *testing.T) {
		vaultPath := setupTestVault(t, "stats-empty")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		cmd := dedupStatsCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed: %v", err)
			}
		})

		if !strings.Contains(output, "Deduplication Statistics:") {
			t.Error("Stats output should contain statistics header")
		}
		if !strings.Contains(output, "Total chunks:") {
			t.Error("Stats output should contain total chunks info")
		}
		if !strings.Contains(output, "Total size:") {
			t.Error("Stats output should contain total size info")
		}
		if !strings.Contains(output, "Space saved:") {
			t.Error("Stats output should contain space saved info")
		}
	})

	t.Run("StatsWithDisabledDeduplication", func(t *testing.T) {
		vaultPath := setupTestVault(t, "stats-disabled")

		// Create config with disabled deduplication
		vaultConfig := &config.VaultConfig{
			Deduplication: config.DeduplicationConfig{
				Enabled: false,
			},
		}
		err := config.SaveVaultConfig(vaultPath, vaultConfig)
		if err != nil {
			t.Fatalf("Failed to save vault config: %v", err)
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

		cmd := dedupStatsCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Stats command failed: %v", err)
			}
		})

		if !strings.Contains(output, "Deduplication enabled: false") {
			t.Error("Stats output should show deduplication disabled")
		}
	})

	t.Run("StatsOutsideVault", func(t *testing.T) {
		tempDir := testutil.TempDir(t, "non-vault-stats")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		cmd := dedupStatsCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when running stats outside vault")
			}
		})

		if !strings.Contains(output, "not inside a vault") {
			t.Error("Error output should indicate not inside a vault")
		}
	})
}

func TestDedupGCCommand(t *testing.T) {
	t.Run("GCInEmptyVault", func(t *testing.T) {
		vaultPath := setupTestVault(t, "gc-empty")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		cmd := dedupGcCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("GC command failed: %v", err)
			}
		})

		if !strings.Contains(output, "Running garbage collection...") {
			t.Error("GC output should contain progress message")
		}
		if !strings.Contains(output, "Garbage collection completed") {
			t.Error("GC output should contain completion message")
		}
	})

	t.Run("GCWithDisabledDeduplication", func(t *testing.T) {
		vaultPath := setupTestVault(t, "gc-disabled")

		// Create config with disabled deduplication
		vaultConfig := &config.VaultConfig{
			Deduplication: config.DeduplicationConfig{
				Enabled: false,
			},
		}
		err := config.SaveVaultConfig(vaultPath, vaultConfig)
		if err != nil {
			t.Fatalf("Failed to save vault config: %v", err)
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

		cmd := dedupGcCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when running GC with disabled deduplication")
			}
		})

		if !strings.Contains(output, "deduplication is not enabled") {
			t.Error("Error output should indicate deduplication is not enabled")
		}
	})
}

func TestDedupOptimizeCommand(t *testing.T) {
	t.Run("OptimizeInEmptyVault", func(t *testing.T) {
		vaultPath := setupTestVault(t, "optimize-empty")

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)
		err = os.Chdir(vaultPath)
		if err != nil {
			t.Fatalf("Failed to change to vault directory: %v", err)
		}

		cmd := dedupOptimizeCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Optimize command failed: %v", err)
			}
		})

		if !strings.Contains(output, "Optimizing vault storage...") {
			t.Error("Optimize output should contain progress message")
		}
		if !strings.Contains(output, "Optimization Results:") {
			t.Error("Optimize output should contain results header")
		}
	})

	t.Run("OptimizeWithDisabledDeduplication", func(t *testing.T) {
		vaultPath := setupTestVault(t, "optimize-disabled")

		// Create config with disabled deduplication
		vaultConfig := &config.VaultConfig{
			Deduplication: config.DeduplicationConfig{
				Enabled: false,
			},
		}
		err := config.SaveVaultConfig(vaultPath, vaultConfig)
		if err != nil {
			t.Fatalf("Failed to save vault config: %v", err)
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

		cmd := dedupOptimizeCmd
		cmd.SetArgs([]string{})

		output := captureOutput(func() {
			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when running optimize with disabled deduplication")
			}
		})

		if !strings.Contains(output, "deduplication is not enabled") {
			t.Error("Error output should indicate deduplication is not enabled")
		}
	})
}

func TestDedupCommandsWithData(t *testing.T) {
	t.Run("CommandsWithPopulatedVault", func(t *testing.T) {
		vaultPath := setupTestVault(t, "populated-vault")

		// Create some test chunks to simulate a vault with data
		chunksDir := filepath.Join(vaultPath, ".sietch", "chunks")
		err := os.MkdirAll(chunksDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create chunks directory: %v", err)
		}

		// Create some dummy chunk files
		for i := 0; i < 5; i++ {
			chunkFile := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d.dat", i))
			err := os.WriteFile(chunkFile, []byte(fmt.Sprintf("chunk data %d", i)), 0o644)
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

		// Test stats command
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

		// Test gc command
		gcCmd := dedupGcCmd
		gcCmd.SetArgs([]string{})

		gcOutput := captureOutput(func() {
			err := gcCmd.Execute()
			if err != nil {
				t.Errorf("GC command failed: %v", err)
			}
		})

		if !strings.Contains(gcOutput, "Garbage collection completed") {
			t.Error("GC output should contain completion message")
		}

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
			t.Error("Optimize output should contain results header")
		}
	})
}

// Helper functions

func setupTestVault(t *testing.T, name string) string {
	vaultPath := testutil.TempDir(t, name)

	// Create vault structure
	err := os.MkdirAll(filepath.Join(vaultPath, ".sietch", "chunks"), 0o755)
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

	err = config.SaveVaultConfig(vaultPath, vaultConfig)
	if err != nil {
		t.Fatalf("Failed to save vault config: %v", err)
	}

	return vaultPath
}

func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	outC := make(chan string)
	go func() {
		var buf strings.Builder
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	out := <-outC

	return out
}