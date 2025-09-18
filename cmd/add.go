/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/util"
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

		// Check if file exists and that it is not a directory or symlink
		fileInfo, err := fs.VerifyFileAndReturnFileInfo(filePath)
		if err != nil {
			return err
		}

		// Load vault configuration
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		// Parse chunk size
		chunkSize, err := util.ParseChunkSize(vaultConfig.Chunking.ChunkSize)
		if err != nil {
			// Fallback to default if parsing fails
			fmt.Printf("Warning: Invalid chunk size in configuration (%s). Using default (4MB).\n",
				vaultConfig.Chunking.ChunkSize)
			chunkSize = int64(constants.DefaultChunkSize) // Default to 4MB
		}

		// Get file size in human-readable format
		sizeInBytes := fileInfo.Size()
		sizeReadable := util.HumanReadableSize(sizeInBytes)

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
		fmt.Println("\nBeginning chunking process...")

		// Get passphrase if needed for encryption
		passphrase, err := ui.GetPassphraseForVault(cmd, vaultConfig)
		if err != nil {
			return err
		}

		// Process the file and store chunks - using the appropriate chunking function
		var chunkRefs []config.ChunkRef
		chunkRefs, err = chunk.ChunkFile(filePath, chunkSize, vaultRoot, passphrase)

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
			Tags:        tags, // Include tags in the manifest
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

func init() {
	rootCmd.AddCommand(addCmd)

	// Optional flags for the add command
	addCmd.Flags().BoolP("force", "f", false, "Force add without confirmation")
	addCmd.Flags().StringP("tags", "t", "", "Comma-separated tags to associate with the file")
	addCmd.Flags().StringP("passphrase-value", "p", "", "Passphrase for encrypted vault (if required)")
}

//TODO: Add support for directories and symlinks
//TODO: Need to check how symlinks will be handled
//TODO: Multiple file support - sietch add file1 file2
//TODO: Interactive mode with real time progress indicators
