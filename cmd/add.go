/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <source_path> <destination_path>",
	Short: "Add a file to the Sietch vault",
	Long: `Add a file to your Sietch vault.

This command adds a file from the specified source path to the destination
path in your vault, then processes it according to your vault configuration.

Example:
  sietch add document.txt vault/documents/
  sietch add ~/photos/vacation.jpg vault/photos/`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		destPath := args[1]

		// Get tags from flags
		tagsFlag, err := cmd.Flags().GetString("tags")
		if err != nil {
			return fmt.Errorf("error parsing tags flag: %v", err)
		}

		tags := []string{}
		if tagsFlag != "" {
			tags = strings.Split(tagsFlag, ",")
		}

		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}
		// Check if vault is initialized
		if !fs.IsVaultInitialized(vaultRoot) {
			return fmt.Errorf("vault not initialized, run 'sietch init' first")
		}

		// Check if file exists
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", filePath)
			}
			return fmt.Errorf("error accessing file: %v", err)
		}

		// Verify it's a regular file, not a directory or symlink
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", filePath)
		}

		// Get file size in human-readable format
		sizeInBytes := fileInfo.Size()
		sizeReadable := humanReadableSize(sizeInBytes)

		// Display file metadata for confirmation
		fmt.Printf("\nFile Metadata:\n")
		fmt.Printf("File: %s\n", filepath.Base(filePath))
		fmt.Printf("Source: %s\n", filePath)
		fmt.Printf("Size: %s (%d bytes)\n", sizeReadable, sizeInBytes)
		fmt.Printf("Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))
		fmt.Printf("Destination: %s\n", destPath)
		if len(tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
		}

		// Process the file in chunks (2MB = 2 * 1024 * 1024 bytes)
		// For real implementation, use 4MB (4 * 1024 * 1024)
		chunkSize := int64(2 * 1024 * 1024)
		fmt.Println("\nBeginning chunking process...")

		// Process the file and store chunks
		chunkRefs, err := chunk.ChunkFile(filePath, chunkSize, vaultRoot)
		if err != nil {
			return fmt.Errorf("chunking failed: %v", err)
		}

		// Create and store the file manifest
		fileManifest := &config.FileManifest{
			FilePath:    filepath.Base(filePath),
			Size:        sizeInBytes,
			ModTime:     fileInfo.ModTime().Format(time.RFC3339),
			Chunks:      chunkRefs,
			Destination: destPath,
			AddedAt:     time.Now().UTC(),
		}

		// Save the manifest
		err = manifest.StoreFileManifest(vaultRoot, filepath.Base(filePath), fileManifest)
		if err != nil {
			return fmt.Errorf("failed to store manifest: %v", err)
		}

		fmt.Printf("\nChunking completed successfully\n")
		fmt.Printf("✓ File added to vault: %s\n", filepath.Base(filePath))
		fmt.Printf("✓ %d chunks stored in vault\n", len(chunkRefs))
		fmt.Printf("✓ Manifest written to .sietch/manifests/%s.yaml\n", filepath.Base(filePath))

		return nil
	},
}

// humanReadableSize converts bytes to a human-readable size string
func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Optional flags for the add command
	addCmd.Flags().BoolP("force", "f", false, "Force add without confirmation")
	addCmd.Flags().StringP("tags", "t", "", "Comma-separated tags to associate with the file")
}
