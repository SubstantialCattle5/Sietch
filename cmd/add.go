/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/atomic"
	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/fs"

	// manifest raw storage removed in favor of transactional helper
	"github.com/substantialcattle5/sietch/internal/progress"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/util"
	"gopkg.in/yaml.v3"
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

		// Get recursive and includeHidden flags
		recursive, _ := cmd.Flags().GetBool("recursive")
		includeHidden, _ := cmd.Flags().GetBool("include-hidden")

		// Expand directories if needed
		filePairs, err = expandDirectories(filePairs, recursive, includeHidden)
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

		// Begin transaction encompassing entire add set for atomicity
		// vaultRoot already resolved earlier; reuse variable
		txn, err := atomic.Begin(vaultRoot, map[string]any{"command": "add", "fileCount": len(filePairs)})
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = txn.Rollback()
				fmt.Println("txn rollback; add operation did not complete")
			}
		}()

		for i, pair := range filePairs {
			// Enhanced progress display for multiple files
			if len(filePairs) > 1 {
				fmt.Printf("[%d/%d] Processing: %s → %s\n",
					i+1, len(filePairs), filepath.Base(pair.Source), pair.Destination)
			} else {
				fmt.Printf("Processing: %s\n", pair.Source)
			}

			// Determine path type and handle accordingly
			fileInfo, pathType, err := fs.GetPathInfo(pair.Source)
			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Handle different path types
			var actualSourcePath string
			switch pathType {
			case fs.PathTypeFile:
				// Regular file - use as is
				actualSourcePath = pair.Source

			case fs.PathTypeSymlink:
				// Resolve symlink and verify target is a regular file
				targetPath, targetInfo, targetType, err := fs.ResolveSymlink(pair.Source)
				if err != nil {
					errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				if targetType != fs.PathTypeFile {
					errorMsg := fmt.Sprintf("✗ %s: symlink target is not a regular file", filepath.Base(pair.Source))
					fmt.Println(errorMsg)
					failedFiles = append(failedFiles, errorMsg)
					continue
				}

				// Use the resolved target path for processing
				actualSourcePath = targetPath
				fileInfo = targetInfo

				if verbose {
					fmt.Printf("  Resolved symlink: %s → %s\n", pair.Source, targetPath)
				}

			case fs.PathTypeDir:
				// Directories should have been expanded already
				errorMsg := fmt.Sprintf("✗ %s: unexpected directory in processing loop", filepath.Base(pair.Source))
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue

			default:
				errorMsg := fmt.Sprintf("✗ %s: unsupported file type", filepath.Base(pair.Source))
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Get file size in human-readable format
			sizeInBytes := fileInfo.Size()
			sizeReadable := util.HumanReadableSize(sizeInBytes)

			// Compute content hash for the entire file
			contentHash, err := computeContentHash(actualSourcePath, vaultConfig.Chunking.HashAlgorithm)
			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

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
			// Use transactional chunking to stage new chunks
			chunkRefs, err = chunk.ChunkFileTransactional(ctx, actualSourcePath, chunkSize, vaultRoot, passphrase, progressMgr, txn)

			if err != nil {
				errorMsg := fmt.Sprintf("✗ %s: chunking failed - %v", filepath.Base(pair.Source), err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Create and store the file manifest
			// Separate directory from filename in destination
			// If pair.Destination ends with a filename (from directory expansion),
			// we need to extract the directory part and filename separately
			destDir := filepath.Dir(pair.Destination)
			destFileName := filepath.Base(pair.Destination)

			// If the destination is just a filename (no directory), set destDir to empty
			if destDir == "." {
				destDir = ""
			} else if destDir != "" && !strings.HasSuffix(destDir, "/") {
				destDir = destDir + "/"
			}

			fileManifest := &config.FileManifest{
				FilePath:    destFileName,
				Size:        sizeInBytes,
				ModTime:     fileInfo.ModTime().Format(time.RFC3339),
				Chunks:      chunkRefs,
				Destination: destDir,
				ContentHash: contentHash,
				AddedAt:     time.Now().UTC(),
				Tags:        tags, // Include tags in the manifest
			}

			// Save the manifest
			// Store manifest via transaction (stage create)
			if err := storeManifestTransactional(txn, vaultRoot, filepath.Base(pair.Source), fileManifest); err != nil {
				if err.Error() == "skipped" {
					errorMsg := fmt.Sprintf("✗ '%s': skipped", fileManifest.Destination+fileManifest.FilePath)
					fmt.Println(errorMsg)
					continue
				}
				errorMsg := fmt.Sprintf("✗ %s: manifest storage failed - %v", fileManifest.FilePath, err)
				fmt.Println(errorMsg)
				failedFiles = append(failedFiles, errorMsg)
				continue
			}

			// Calculate space savings for this file
			spaceSavings := calculateSpaceSavings(chunkRefs)

			// Success message
			if len(filePairs) > 1 {
				fmt.Printf("✓ %s (%d chunks", filepath.Base(pair.Source), len(chunkRefs))
				if spaceSavings.SpaceSaved > 0 {
					fmt.Printf(", %s saved", util.HumanReadableSize(spaceSavings.SpaceSaved))
				}
				fmt.Printf(")\n")
			} else {
				fmt.Printf("✓ File added to vault: %s\n", filepath.Base(pair.Source))
				fmt.Printf("✓ %d chunks stored in vault\n", len(chunkRefs))
				if spaceSavings.SpaceSaved > 0 {
					fmt.Printf("✓ Space saved: %s (%.1f%%)\n",
						util.HumanReadableSize(spaceSavings.SpaceSaved),
						spaceSavings.SpaceSavedPct)
				}
				fmt.Printf("✓ Manifest written to .sietch/manifests/%s.yaml\n", filepath.Base(pair.Source))
			}

			successCount++

			// Add to total space savings
			fileSavings := calculateSpaceSavings(chunkRefs)
			totalSpaceSavings.OriginalSize += fileSavings.OriginalSize
			totalSpaceSavings.CompressedSize += fileSavings.CompressedSize
			totalSpaceSavings.SpaceSaved += fileSavings.SpaceSaved
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

		// Commit transaction if we had any successes
		if successCount == 0 {
			return fmt.Errorf("all files failed to process")
		}
		if err := txn.Commit(); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}
		committed = true
		fmt.Println("txn successful; add committed")
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

// expandDirectories expands directories into file pairs if recursive flag is set
func expandDirectories(pairs []FilePair, recursive bool, includeHidden bool) ([]FilePair, error) {
	var expandedPairs []FilePair

	for _, pair := range pairs {
		// Get path info to determine type
		fileInfo, pathType, err := fs.GetPathInfo(pair.Source)
		if err != nil {
			return nil, err
		}

		switch pathType {
		case fs.PathTypeFile:
			// Regular file - add as is
			expandedPairs = append(expandedPairs, pair)

		case fs.PathTypeSymlink:
			// Symlink - will be handled in processing loop, add as is
			expandedPairs = append(expandedPairs, pair)

		case fs.PathTypeDir:
			// Directory - expand if recursive, otherwise error
			if !recursive {
				return nil, fmt.Errorf("'%s' is a directory. Use --recursive flag to add directories", pair.Source)
			}

			// Walk the directory tree
			err := filepath.WalkDir(pair.Source, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip hidden files/directories if includeHidden is false
				if fs.ShouldSkipHidden(d.Name(), includeHidden) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Only add regular files and symlinks
				if !d.IsDir() {
					// Compute relative path from source directory
					relPath, err := filepath.Rel(pair.Source, path)
					if err != nil {
						return fmt.Errorf("failed to compute relative path: %v", err)
					}

					// Preserve directory structure in destination
					destPath := filepath.Join(pair.Destination, relPath)

					expandedPairs = append(expandedPairs, FilePair{
						Source:      path,
						Destination: destPath,
					})
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("error walking directory '%s': %v", pair.Source, err)
			}

		default:
			return nil, fmt.Errorf("'%s' is not a regular file, directory, or symlink", pair.Source)
		}

		_ = fileInfo // fileInfo might be used for verbose output later
	}

	return expandedPairs, nil
}

// computeContentHash computes the hash of the entire file content
func computeContentHash(filePath, hashAlgorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %v", err)
	}
	defer file.Close()

	hasher, err := chunk.CreateHasher(hashAlgorithm)
	if err != nil {
		return "", fmt.Errorf("failed to create hasher: %v", err)
	}

	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to hash file content: %v", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Optional flags for the add command
	addCmd.Flags().BoolP("force", "f", false, "Force add without confirmation")
	addCmd.Flags().StringP("tags", "t", "", "Comma-separated tags to associate with the file")
	addCmd.Flags().BoolP("recursive", "r", false, "Recursively add directories")
	addCmd.Flags().BoolP("include-hidden", "H", false, "Include hidden files and directories")
	addCmd.Flags().Bool("passphrase-stdin", false, "Read passphrase from stdin (for automation)")
	addCmd.Flags().String("passphrase-file", "", "Read passphrase from file (file should have 0600 permissions)")
}

// storeManifestTransactional writes a manifest yaml via the transaction staging new file.
func storeManifestTransactional(txn *atomic.Transaction, vaultRoot string, fileName string, m *config.FileManifest) error {
	// Mirror logic from manifest.StoreFileManifest but stage instead of direct write.
	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create manifests directory: %v", err)
	}
	destination := strings.ReplaceAll(m.Destination, "/", ".")
	uniqueFileIdentifier := destination + fileName + ".yaml"
	relPath := filepath.ToSlash(filepath.Join(".sietch", "manifests", uniqueFileIdentifier))
	// Prompt overwrite if exists in final location
	finalPath := filepath.Join(manifestsDir, uniqueFileIdentifier)
	if _, err := os.Stat(finalPath); err == nil {
		message := fmt.Sprintf("'%s' exists. Overwrite? ", m.Destination+fileName)
		response, err2 := util.ConfirmOverwrite(message, os.Stdin, os.Stdout)
		if err2 != nil || !response {
			return fmt.Errorf("skipped")
		}
		// Stage replace instead of create
		w, err2 := txn.StageReplace(relPath)
		if err2 != nil {
			return err2
		}
		defer w.Close()
		return writeManifestYAML(w, m)
	}
	w, err := txn.StageCreate(relPath)
	if err != nil {
		return err
	}
	defer w.Close()
	return writeManifestYAML(w, m)
}

func writeManifestYAML(w io.Writer, m *config.FileManifest) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return nil
}

//TODO: Need to check how symlinks will be handled
//TODO: Interactive mode with real time progress indicators
