/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/internal/progress"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/util"
)

// SpaceSavings represents space savings statistics for a file
type SpaceSavings struct {
	OriginalSize   int64
	CompressedSize int64
	SpaceSaved     int64
	SpaceSavedPct  float64
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <source_path> <destination_path> [source_path2] [destination_path2]...",
	Short: "Add files, directories, or symlinks to the Sietch vault",
	Long: `Add files, directories, or symlinks to your Sietch vault.

This command adds files from the specified source paths to the destination
paths in your vault. It supports regular files, directories (recursively),
and symbolic links (by copying their contents).

Supports two usage patterns:
1. Paired arguments: sietch add source1 dest1 source2 dest2 ...
	  Each source is stored at its corresponding destination path.

2. Single destination: sietch add source1 source2 ... dest
	  All sources are stored under the same destination directory.

Directory handling:
- Directories are processed recursively
- Directory structure is preserved in the destination
- Hidden files and directories are included
- Symlinks within directories are followed

Symlink handling:
- Symlinks to files: the target file content is added
- Symlinks to directories: all files in the target directory are added recursively

Examples:
	 sietch add document.txt vault/documents/
	 sietch add file1.txt dest1/ file2.txt dest2/
	 sietch add ~/photos vault/photos/
	 sietch add ~/link-to-file.txt vault/files/
	 sietch add ~/photos/img1.jpg ~/docs/ vault/backup/`,
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

		// Get global flags
		verbose, _ := cmd.Flags().GetBool("verbose")
		quiet, _ := cmd.Flags().GetBool("quiet")

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

		// Create progress manager
		progressMgr := progress.NewManager(progress.Options{
			Quiet:   quiet,
			Verbose: verbose,
		})

		// Create context with cancellation
		ctx := context.Background()
		ctx = progressMgr.SetupCancellation(ctx)

		// Process each file pair
		successCount := 0
		var failedFiles []string
		var totalSpaceSavings SpaceSavings

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

			// Check if path exists (accepts files, directories, and symlinks)
			pathInfo, err := fs.VerifyPathAndReturnInfo(pair.Source)
			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Collect all files to process (handles directories and symlinks)
			var filesToProcess []string
			if pathInfo.IsDir() || pathInfo.Mode()&os.ModeSymlink != 0 {
				// For directories and symlinks, recursively collect all files
				filesToProcess, err = fs.CollectFilesRecursively(pair.Source)
				if err != nil {
					errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				if len(filesToProcess) == 0 {
					fmt.Printf("⚠ %s: directory is empty, skipping\n", filepath.Base(pair.Source))
					continue
				}

				fmt.Printf("  Found %d file(s) to add\n", len(filesToProcess))
			} else {
				// Regular file
				filesToProcess = []string{pair.Source}
			}

			// Process each collected file
			processedCount := 0
			for _, sourceFile := range filesToProcess {
				// Get file info for the actual file
				fileInfo, err := fs.VerifyFileAndReturnFileInfo(sourceFile)
				if err != nil {
					errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(sourceFile), err)
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				// Get file size in human-readable format
				sizeInBytes := fileInfo.Size()
				sizeReadable := util.HumanReadableSize(sizeInBytes)

				// Display file metadata for confirmation (only for single files or when verbose)
				if len(filePairs) == 1 || verbose {
					fmt.Printf("    %s: %s (%d bytes)\n", filepath.Base(sourceFile), sizeReadable, sizeInBytes)
					if verbose {
						fmt.Printf("    Modified: %s\n", fileInfo.ModTime().Format(time.RFC3339))
					}
				}

				// Calculate relative path for destination if processing directory
				var destPath string
				if len(filesToProcess) > 1 {
					// For directories, preserve the structure
					relPath, err := filepath.Rel(pair.Source, sourceFile)
					if err != nil {
						// If we can't get relative path, use just the destination directory
						destPath = pair.Destination
					} else {
						// Destination should be directory only (excluding the filename)
						// The FilePath field in manifest will have the filename
						destPath = filepath.Join(pair.Destination, filepath.Dir(relPath))
						// Ensure it ends with / for directory paths
						if destPath != "" && !strings.HasSuffix(destPath, "/") {
							destPath += "/"
						}
					}
				} else {
					// For single files, destination is the directory
					destPath = pair.Destination
				}

				// Process the file and store chunks
				var chunkRefs []config.ChunkRef
				chunkRefs, err = chunk.ChunkFile(ctx, sourceFile, chunkSize, vaultRoot, passphrase, progressMgr)

				if err != nil {
					errorMsg := fmt.Sprintf("✗ %s: chunking failed - %v", filepath.Base(sourceFile), err)
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				// Create and store the file manifest
				fileManifest := &config.FileManifest{
					FilePath:    filepath.Base(sourceFile),
					Size:        sizeInBytes,
					ModTime:     fileInfo.ModTime().Format(time.RFC3339),
					Chunks:      chunkRefs,
					Destination: destPath,
					AddedAt:     time.Now().UTC(),
					Tags:        tags, // Include tags in the manifest
				}

				// Save the manifest
				err = manifest.StoreFileManifest(vaultRoot, filepath.Base(sourceFile), fileManifest)
				if err != nil {
					errorMsg := fmt.Sprintf("✗ %s: manifest storage failed - %v", filepath.Base(sourceFile), err)
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				// Calculate space savings for this file
				fileSavings := calculateSpaceSavings(chunkRefs)
				totalSpaceSavings.OriginalSize += fileSavings.OriginalSize
				totalSpaceSavings.CompressedSize += fileSavings.CompressedSize
				totalSpaceSavings.SpaceSaved += fileSavings.SpaceSaved

				// Success message (compact for directories)
				if len(filesToProcess) > 1 {
					fmt.Printf("    ✓ %s (%d chunks)\n", filepath.Base(sourceFile), len(chunkRefs))
				} else if len(filePairs) > 1 {
					fmt.Printf("✓ %s (%d chunks", filepath.Base(sourceFile), len(chunkRefs))
					if fileSavings.SpaceSaved > 0 {
						fmt.Printf(", %s saved", util.HumanReadableSize(fileSavings.SpaceSaved))
					}
					fmt.Printf(")\n")
				} else {
					fmt.Printf("✓ File added to vault: %s\n", filepath.Base(sourceFile))
					fmt.Printf("✓ %d chunks stored in vault\n", len(chunkRefs))
					if fileSavings.SpaceSaved > 0 {
						fmt.Printf("✓ Space saved: %s (%.1f%%)\n",
							util.HumanReadableSize(fileSavings.SpaceSaved),
							fileSavings.SpaceSavedPct)
					}
					fmt.Printf("✓ Manifest written to .sietch/manifests/%s.yaml\n", filepath.Base(sourceFile))
				}

				processedCount++
			}

			// Update success count for this pair
			if processedCount > 0 {
				successCount += processedCount
				if len(filesToProcess) > 1 {
					fmt.Printf("✓ Added %d file(s) from %s\n", processedCount, filepath.Base(pair.Source))
				}
			}
		}

		// Cleanup progress manager
		progressMgr.Cleanup()

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

			// Show vault configuration details
			fmt.Printf("\n📋 Vault Configuration:\n")
			fmt.Printf("  • Encryption: %s", vaultConfig.Encryption.Type)
			if vaultConfig.Encryption.PassphraseProtected {
				fmt.Printf(" (passphrase protected)")
			}
			fmt.Println()

			fmt.Printf("  • Compression: %s\n", vaultConfig.Compression)

			fmt.Printf("  • Chunking: %s (size: %s)\n", vaultConfig.Chunking.Strategy, vaultConfig.Chunking.ChunkSize)

			// Show total space savings if compression is used
			if vaultConfig.Compression != "none" && totalSpaceSavings.SpaceSaved > 0 {
				totalSpaceSavedPct := float64(0)
				if totalSpaceSavings.OriginalSize > 0 {
					totalSpaceSavedPct = float64(totalSpaceSavings.SpaceSaved) / float64(totalSpaceSavings.OriginalSize) * 100
				}
				fmt.Printf("\n💾 Total Space Savings:\n")
				fmt.Printf("  • Original size: %s\n", util.HumanReadableSize(totalSpaceSavings.OriginalSize))
				fmt.Printf("  • Compressed size: %s\n", util.HumanReadableSize(totalSpaceSavings.CompressedSize))
				fmt.Printf("  • Space saved: %s (%.1f%%)\n",
					util.HumanReadableSize(totalSpaceSavings.SpaceSaved),
					totalSpaceSavedPct)
			}
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

// calculateSpaceSavings calculates space savings for a file based on its chunks
func calculateSpaceSavings(chunks []config.ChunkRef) SpaceSavings {
	originalSize := int64(0)
	compressedSize := int64(0)

	for _, chunk := range chunks {
		originalSize += chunk.Size
		if chunk.CompressedSize > 0 {
			compressedSize += chunk.CompressedSize
		} else {
			// If no compressed size is recorded, use original size
			compressedSize += chunk.Size
		}
	}

	spaceSaved := originalSize - compressedSize
	var spaceSavedPct float64
	if originalSize > 0 {
		spaceSavedPct = float64(spaceSaved) / float64(originalSize) * 100
	}

	return SpaceSavings{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		SpaceSaved:     spaceSaved,
		SpaceSavedPct:  spaceSavedPct,
	}
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

//TODO: Interactive mode with real time progress indicators
