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
	Use:   "add <source_path> <destination_path> [source_path2] [destination_path2]...",
	Short: "Add one or more files to the Sietch vault",
	Long: `Add multiple files to your Sietch vault.

This command adds files from the specified source paths to the destination
paths in your vault, then processes them according to your vault configuration.

Supports two usage patterns:
1. Paired arguments: sietch add source1 dest1 source2 dest2 ...
	  Each source file is stored at its corresponding destination path.

2. Single destination: sietch add source1 source2 ... dest
	  All source files are stored under the same destination directory.

Examples:
	 sietch add document.txt vault/documents/
	 sietch add file1.txt dest1/ file2.txt dest2/
	 sietch add ~/photos/img1.jpg ~/photos/img2.jpg vault/photos/`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate argument count (reasonable limit for batch operations)
		if len(args) > 100 {
			return fmt.Errorf("too many arguments: maximum 100 files per command (received %d)", len(args))
		}

		// Parse file pairs from arguments
		filePairs, err := parseFileArguments(args)
		if err != nil {
			return err
		}

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

		// Get passphrase if needed for encryption
		passphrase, err := ui.GetPassphraseForVault(cmd, vaultConfig)
		if err != nil {
			return err
		}

		// Process each file pair
		successCount := 0
		var failedFiles []string

		// Show initial progress for multiple files
		if len(filePairs) > 1 {
			fmt.Printf("Starting batch processing of %d files...\n\n", len(filePairs))
		}

		for i, pair := range filePairs {
			// Enhanced progress display for multiple files
			if len(filePairs) > 1 {
				fmt.Printf("[%d/%d] Processing: %s → %s\n",
					i+1, len(filePairs), filepath.Base(pair.Source), pair.Destination)
			} else {
				fmt.Printf("Processing: %s\n", pair.Source)
			}

			// Check if file exists and that it is not a directory or symlink
			fileInfo, err := fs.VerifyFileAndReturnFileInfo(pair.Source)
			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Get file size in human-readable format
			sizeInBytes := fileInfo.Size()
			sizeReadable := util.HumanReadableSize(sizeInBytes)

			// Display file metadata for confirmation (only for single files or when verbose)
			verbose, _ := cmd.Flags().GetBool("verbose")
			if len(filePairs) == 1 || verbose {
				fmt.Printf("  Size: %s (%d bytes)\n", sizeReadable, sizeInBytes)
				fmt.Printf("  Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))
				if len(tags) > 0 {
					fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
				}
			}

			// Process the file and store chunks - using the appropriate chunking function
			var chunkRefs []config.ChunkRef
			chunkRefs, err = chunk.ChunkFile(pair.Source, chunkSize, vaultRoot, passphrase)

			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: chunking failed - %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Create and store the file manifest
			fileManifest := &config.FileManifest{
				FilePath:    filepath.Base(pair.Source),
				Size:        sizeInBytes,
				ModTime:     fileInfo.ModTime().Format(time.RFC3339),
				Chunks:      chunkRefs,
				Destination: pair.Destination,
				AddedAt:     time.Now().UTC(),
				Tags:        tags, // Include tags in the manifest
			}

			// Save the manifest
			err = manifest.StoreFileManifest(vaultRoot, filepath.Base(pair.Source), fileManifest)
			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: manifest storage failed - %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Success message
			if len(filePairs) > 1 {
				fmt.Printf("✓ %s (%d chunks)\n", filepath.Base(pair.Source), len(chunkRefs))
			} else {
				fmt.Printf("✓ File added to vault: %s\n", filepath.Base(pair.Source))
				fmt.Printf("✓ %d chunks stored in vault\n", len(chunkRefs))
				fmt.Printf("✓ Manifest written to .sietch/manifests/%s.yaml\n", filepath.Base(pair.Source))
			}

			successCount++
		}

		// Enhanced summary
		fmt.Printf("\n=== Batch Processing Summary ===\n")
		fmt.Printf("Total files: %d\n", len(filePairs))
		fmt.Printf("Successful: %d\n", successCount)

		if len(failedFiles) > 0 {
			fmt.Printf("Failed: %d\n", len(failedFiles))
			if len(failedFiles) <= 5 {
				fmt.Printf("\nFailed files:\n")
				for _, failed := range failedFiles {
					fmt.Printf("  %s\n", failed)
				}
			} else {
				fmt.Printf("\nFirst 5 failed files:\n")
				for i := 0; i < 5; i++ {
					fmt.Printf("  %s\n", failedFiles[i])
				}
				fmt.Printf("  ... and %d more\n", len(failedFiles)-5)
			}
		}

		if successCount > 0 {
			fmt.Printf("\n✓ %d file(s) successfully added to vault\n", successCount)
		}

		// Return error only if all files failed
		if successCount == 0 {
			return fmt.Errorf("all files failed to process")
		}

		return nil
	},
}

// FilePair represents a source file and its destination path
type FilePair struct {
	Source      string
	Destination string
}

// parseFileArguments parses command line arguments into source-destination pairs
// Supports two patterns:
// 1. Paired: source1 dest1 source2 dest2 ... (even number of args)
// 2. Single destination: source1 source2 ... dest (odd number of args, last is dest)
func parseFileArguments(args []string) ([]FilePair, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("must provide at least one source and one destination")
	}

	// Check if even number of arguments (paired pattern)
	if len(args)%2 == 0 {
		// Paired pattern: source1 dest1 source2 dest2 ...
		var pairs []FilePair
		for i := 0; i < len(args); i += 2 {
			pairs = append(pairs, FilePair{
				Source:      args[i],
				Destination: args[i+1],
			})
		}
		return pairs, nil
	}

	// Odd number of arguments (single destination pattern)
	// Last argument is the destination for all sources
	destination := args[len(args)-1]
	var pairs []FilePair

	for i := 0; i < len(args)-1; i++ {
		pairs = append(pairs, FilePair{
			Source:      args[i],
			Destination: destination,
		})
	}

	return pairs, nil
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
//TODO: Interactive mode with real time progress indicators
