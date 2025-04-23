/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <file_path>",
	Short: "Delete a file from the Sietch vault",
	Long: `Delete a file from your Sietch vault.

This command removes a file from your vault and cleans up any orphaned
chunks that are no longer referenced by other files.

Examples:
  sietch delete docs/report.pdf        # Delete a specific file
  sietch delete --force notes.txt      # Delete without confirmation
  sietch delete --keep-chunks photo.jpg # Delete manifest but keep chunks`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		// Find vault root
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Create a vault manager
		manager, err := config.NewManager(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to create vault manager: %v", err)
		}

		// Get the vault manifest to find the file
		manifest, err := manager.GetManifest()
		if err != nil {
			return fmt.Errorf("failed to get vault manifest: %v", err)
		}

		// Find the file in the manifest
		var targetFile *config.FileManifest
		var fileBaseName string

		for _, file := range manifest.Files {
			fullPath := file.Destination + file.FilePath
			if fullPath == filePath || file.FilePath == filePath {
				targetFile = &file
				fileBaseName = file.FilePath
				break
			}
		}

		if targetFile == nil {
			return fmt.Errorf("file not found in vault: %s", filePath)
		}

		// Get confirmation unless --force is specified
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", filePath)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Operation cancelled")
				return nil
			}
		}

		// Step 1: Remove the manifest file
		manifestPath := filepath.Join(vaultRoot, ".sietch", "manifests", fileBaseName+".yaml")
		if err := os.Remove(manifestPath); err != nil {
			return fmt.Errorf("failed to remove manifest file: %v", err)
		}

		// Step 2: Clean up orphaned chunks if --keep-chunks is not specified
		keepChunks, _ := cmd.Flags().GetBool("keep-chunks")
		if !keepChunks {
			// Get the remaining manifests to check for chunk references
			remainingManifest, err := manager.GetManifest()
			if err != nil {
				fmt.Printf("Warning: Failed to check for orphaned chunks: %v\n", err)
			} else {
				// Find and remove orphaned chunks
				if err := cleanupOrphanedChunks(vaultRoot, targetFile.Chunks, remainingManifest); err != nil {
					fmt.Printf("Warning: Failed to clean up some orphaned chunks: %v\n", err)
				}
			}
		}

		fmt.Printf("✓ Successfully deleted '%s' from vault\n", filePath)
		return nil
	},
}

// cleanupOrphanedChunks removes chunks that are no longer referenced by any file
func cleanupOrphanedChunks(vaultRoot string, deletedChunks []config.ChunkRef, remainingManifest *config.Manifest) error {
	// Create a map of all chunks still in use
	chunksInUse := make(map[string]bool)
	for _, file := range remainingManifest.Files {
		for _, chunk := range file.Chunks {
			chunksInUse[chunk.Hash] = true
		}
	}

	// Delete chunks that are no longer in use
	deletedCount := 0
	var lastError error

	for _, chunk := range deletedChunks {
		if !chunksInUse[chunk.Hash] {
			// This chunk is not referenced by any other file, safe to delete
			chunkPath := filepath.Join(vaultRoot, ".sietch", "chunks", chunk.Hash)
			if err := os.Remove(chunkPath); err != nil {
				lastError = err
				fmt.Printf("Warning: Failed to delete chunk %s: %v\n", chunk.Hash, err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("✓ Removed %d orphaned chunks\n", deletedCount)
	}

	return lastError
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Add flags
	deleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	deleteCmd.Flags().Bool("keep-chunks", false, "Keep chunks, only delete manifest")
}
