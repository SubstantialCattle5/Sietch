/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/compression"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/progress"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/util"
)

// findFileManifest searches for a file manifest by path, trying multiple approaches
func findFileManifest(vaultRoot, filePath string) (*config.FileManifest, error) {
	// Create a vault manager to get all manifests
	manager, err := config.NewManager(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault manager: %v", err)
	}

	// Get all manifests
	vaultManifest, err := manager.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get vault manifest: %v", err)
	}

	// Search through all files to find a match
	for _, fileManifest := range vaultManifest.Files {
		// Try multiple matching strategies:
		// 1. Exact match with full path (Destination + FilePath)
		fullPath := fileManifest.Destination + fileManifest.FilePath
		if fullPath == filePath {
			return &fileManifest, nil
		}

		// 2. Match just the FilePath
		if fileManifest.FilePath == filePath {
			return &fileManifest, nil
		}

		// 3. Match basename if user provided just filename
		if filepath.Base(fileManifest.FilePath) == filePath {
			return &fileManifest, nil
		}

		// 4. Match basename of full path
		if filepath.Base(fullPath) == filePath {
			return &fileManifest, nil
		}
	}

	// If we get here, no file was found - provide helpful error message
	if len(vaultManifest.Files) == 0 {
		return nil, fmt.Errorf("no files found in vault")
	}

	// Show similar files to help user
	var suggestions []string
	for _, fileManifest := range vaultManifest.Files {
		fullPath := fileManifest.Destination + fileManifest.FilePath
		if filepath.Base(fullPath) == filepath.Base(filePath) {
			suggestions = append(suggestions, fullPath)
		}
	}

	if len(suggestions) > 0 {
		return nil, fmt.Errorf("no file found matching '%s'. Did you mean one of: %v", filePath, suggestions)
	}

	return nil, fmt.Errorf("no file found matching '%s'. Use 'sietch ls' to see available files", filePath)
}

const (
	force          = "force"
	skipDecryption = "skip-decryption"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <file_path> <destination_path>",
	Short: "Retrieve a file from the Sietch vault",
	Long: `Retrieve a file from your Sietch vault.

This command retrieves a file from your vault, decrypts it if necessary,
and writes it to the specified destination.

Example:
  sietch get document.txt ~/Documents/
  sietch get vault/photos/vacation.jpg ./retrieved_photos/`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get global flags
		verbose, _ := cmd.Flags().GetBool("verbose")
		quiet, _ := cmd.Flags().GetBool("quiet")

		// Find vault root
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Load vault configuration to access encryption settings
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		// Parse arguments
		filePath := args[0]
		destPath := "."
		if len(args) > 1 {
			destPath = args[1]
		}

		// Get flags
		force, _ := cmd.Flags().GetBool(force)
		skipEncryption, _ := cmd.Flags().GetBool(skipDecryption)

		if !quiet {
			fmt.Printf("Retrieving %s from vault\n", filePath)
		}

		// Find the file manifest by searching through all manifests
		fileManifest, err := findFileManifest(vaultRoot, filePath)
		if err != nil {
			return fmt.Errorf("file not found in vault: %v", err)
		}

		// Determine output path
		outputPath := filepath.Join(destPath, fileManifest.FilePath)
		if _, err := os.Stat(outputPath); err == nil && !force {
			return fmt.Errorf("file %s already exists, use --force to overwrite", outputPath)
		}

		// Ensure destination directory exists
		destDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("failed to create destination directory: %v", err)
		}

		// Create output file
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outputFile.Close()

		// Get passphrase if needed for decryption
		passphrase, err := ui.GetPassphraseForVault(cmd, vaultConfig)
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %v", err)
		}

		// Initialize deduplication manager
		dedupManager, err := deduplication.NewManager(vaultRoot, vaultConfig.Deduplication)
		if err != nil {
			return fmt.Errorf("failed to initialize deduplication manager: %v", err)
		}

		// Create progress manager
		progressMgr := progress.NewManager(progress.Options{
			Quiet:   quiet,
			Verbose: verbose,
		})

		// Create context with cancellation
		ctx := context.Background()
		ctx = progressMgr.SetupCancellation(ctx)

		// Process each chunk
		chunkCount := len(fileManifest.Chunks)
		totalSize := int64(0)
		for _, chunkRef := range fileManifest.Chunks {
			totalSize += chunkRef.Size
		}

		// Initialize progress bars
		progressMgr.InitTotalProgress(totalSize, "Retrieving file")

		if !quiet {
			fmt.Printf("Reassembling file from %d chunks\n", chunkCount)
		}

		for i, chunkRef := range fileManifest.Chunks {
			// Check for cancellation
			select {
			case <-ctx.Done():
				progressMgr.Cleanup()
				return fmt.Errorf("operation cancelled")
			default:
			}

			progressMgr.PrintVerbose("Processing chunk %d/%d\n", i+1, chunkCount)

			// Get the chunk hash to use - if encrypted, use the encrypted hash
			chunkHash := chunkRef.Hash
			if chunkRef.EncryptedHash != "" {
				chunkHash = chunkRef.EncryptedHash
			}

			// Read the chunk data using deduplication manager
			// This properly resolves chunks through the deduplication index
			chunkData, err := dedupManager.GetChunk(chunkHash)
			if err != nil {
				return fmt.Errorf("failed to read chunk %s: %v", chunkHash, err)
			}

			// Decrypt the chunk if encryption is enabled and not skipped
			if !skipEncryption && vaultConfig.Encryption.Type != "none" {
				if len(chunkData) == 0 {
					return fmt.Errorf("chunk %s is empty", chunkHash)
				}

				// Decrypt the data using the appropriate method based on passphrase protection
				var decryptedData string
				if vaultConfig.Encryption.PassphraseProtected {
					decryptedData, err = encryption.DecryptDataWithPassphrase(
						string(chunkData),
						vaultRoot,
						passphrase,
					)
				} else {
					decryptedData, err = encryption.DecryptData(
						string(chunkData),
						vaultRoot,
					)
				}
				if err != nil {
					return fmt.Errorf("failed to decrypt chunk %s: %v", chunkHash, err)
				}

				// TODO: Implement chunk integrity verification
				// if !skipEncryption && chunkRef.Hash != "" {
				//     if util.SHA256Sum([]byte(decryptedData)) != chunkRef.Hash {
				//         return fmt.Errorf("chunk integrity check failed")
				//     }
				// }

				// The original data was base64-encoded before encryption. Decode back to bytes.
				decodedBytes, err := base64.StdEncoding.DecodeString(decryptedData)
				if err != nil {
					return fmt.Errorf("failed to base64-decode decrypted chunk %s: %v", chunkHash, err)
				}
				chunkData = decodedBytes
			}

			// Decompress the chunk if it was compressed
			if chunkRef.Compressed {
				// Use the compression type stored in the chunk ref, not the current vault config
				// This handles cases where the vault compression setting changed after the file was added
				compressionType := chunkRef.CompressionType
				if compressionType == "" {
					// Fallback to vault config for backwards compatibility with old manifests
					compressionType = vaultConfig.Compression
				}
				decompressedData, err := compression.DecompressData(chunkData, compressionType)
				if err != nil {
					return fmt.Errorf("failed to decompress chunk %s: %v", chunkHash, err)
				}
				chunkData = decompressedData
			}

			// Write the chunk to the output file
			bytesWritten, err := outputFile.Write(chunkData)
			if err != nil {
				progressMgr.Cleanup()
				return fmt.Errorf("failed to write to output file: %v", err)
			}

			// Update progress bars
			progressMgr.UpdateTotalProgress(int64(bytesWritten))
		}

		// Complete progress bars
		progressMgr.FinishTotalProgress()
		progressMgr.Cleanup()

		progressMgr.PrintInfo("\nFile retrieved successfully: %s\n", outputPath)
		progressMgr.PrintInfo("Size: %s\n", util.HumanReadableSize(fileManifest.Size))

		// Show file tags if available
		if len(fileManifest.Tags) > 0 {
			progressMgr.PrintInfo("Tags: %v\n", fileManifest.Tags)
		}

		// Note about encryption status
		if skipEncryption && vaultConfig.Encryption.Type != "none" {
			progressMgr.PrintInfo("\nWarning: File retrieved without decryption (--skip-decryption flag used)")
		} else if vaultConfig.Encryption.Type != "none" {
			progressMgr.PrintInfo("\nFile successfully decrypted")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Add flags
	getCmd.Flags().BoolP(force, "f", false, "Force overwrite if file exists at destination")
	getCmd.Flags().Bool(skipDecryption, false, "Skip decryption and retrieve raw chunks (for recovery)")
}

//TODO: Implement parallel chunk retrieval
//TODO: Implement streaming process
//TODO: Implement atomic operations, rollback on error
