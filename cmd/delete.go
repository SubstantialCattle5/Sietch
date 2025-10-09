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

	"github.com/substantialcattle5/sietch/internal/atomic"
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
				fmt.Println("Operation canceled")
				return nil
			}
		}

		// Begin transaction for delete operation
		txn, err := atomic.Begin(vaultRoot, map[string]any{"command": "delete", "file": filePath})
		if err != nil {
			return fmt.Errorf("begin transaction: %v", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = txn.Rollback()
				fmt.Println("txn rollback; delete operation did not complete")
			}
		}()

		// Step 1: Stage removal of the manifest file
		destination := strings.ReplaceAll(targetFile.Destination, "/", ".")
		uniqueFileIdentifier := destination + fileBaseName + ".yaml"
		// relative manifest path inside vault root
		relManifest := filepath.ToSlash(filepath.Join(".sietch", "manifests", uniqueFileIdentifier))
		if err := txn.StageDelete(relManifest); err != nil {
			return fmt.Errorf("stage manifest delete: %v", err)
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
				if err := stageOrphanedChunkDeletes(txn, vaultRoot, targetFile.Chunks, remainingManifest); err != nil {
					fmt.Printf("Warning: Failed to stage some orphaned chunks: %v\n", err)
				}
			}
		}

		if err := txn.Commit(); err != nil {
			return fmt.Errorf("commit delete transaction: %v", err)
		}
		committed = true
		fmt.Println("txn successful; delete committed")
		fmt.Printf("✓ Successfully deleted '%s' from vault\n", filePath)
		return nil
	},
}

// stageOrphanedChunkDeletes stages deletions for chunks no longer referenced.
func stageOrphanedChunkDeletes(txn *atomic.Transaction, vaultRoot string, deletedChunks []config.ChunkRef, remainingManifest *config.Manifest) error {
	chunksInUse := make(map[string]bool)
	for _, file := range remainingManifest.Files {
		for _, ch := range file.Chunks {
			chunksInUse[ch.Hash] = true
		}
	}
	var lastErr error
	for _, ch := range deletedChunks {
		if chunksInUse[ch.Hash] {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(".sietch", "chunks", ch.Hash))
		if err := txn.StageDelete(rel); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Add flags
	deleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	deleteCmd.Flags().Bool("keep-chunks", false, "Keep chunks, only delete manifest")
}
