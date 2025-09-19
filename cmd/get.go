/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/compression"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
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

		fmt.Printf("Retrieving %s from vault\n", filePath)

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

		// Process each chunk
		chunkCount := len(fileManifest.Chunks)
		fmt.Printf("Reassembling file from %d chunks\n", chunkCount)

		for i, chunkRef := range fileManifest.Chunks {
			fmt.Printf("Processing chunk %d/%d\n", i+1, chunkCount)

			// Get the chunk hash to use - if encrypted, use the encrypted hash
			chunkHash := chunkRef.Hash
			if chunkRef.EncryptedHash != "" {
				chunkHash = chunkRef.EncryptedHash
			}

			// Get the chunk path
			chunkPath := filepath.Join(vaultRoot, ".sietch", "chunks", chunkHash)

			// Check if chunk exists
			if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
				return fmt.Errorf("chunk %s not found", chunkHash)
			}

			// Read the chunk data
			chunkData, err := os.ReadFile(chunkPath)
			if err != nil {
				return fmt.Errorf("failed to read chunk: %v", err)
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

				chunkData = []byte(decryptedData)
			}

			// Decompress the chunk if it was compressed
			if chunkRef.Compressed {
				decompressedData, err := compression.DecompressData(chunkData, vaultConfig.Compression)
				if err != nil {
					return fmt.Errorf("failed to decompress chunk %s: %v", chunkHash, err)
				}
				chunkData = decompressedData
			}

			// Write the chunk to the output file
			_, err = outputFile.Write(chunkData)
			if err != nil {
				return fmt.Errorf("failed to write to output file: %v", err)
			}
		}

		fmt.Printf("\nFile retrieved successfully: %s\n", outputPath)
		fmt.Printf("Size: %s\n", util.HumanReadableSize(fileManifest.Size))

		// Show file tags if available
		if len(fileManifest.Tags) > 0 {
			fmt.Printf("Tags: %v\n", fileManifest.Tags)
		}

		// Note about encryption status
		if skipEncryption && vaultConfig.Encryption.Type != "none" {
			fmt.Println("\nWarning: File retrieved without decryption (--skip-decryption flag used)")
		} else if vaultConfig.Encryption.Type != "none" {
			fmt.Println("\nFile successfully decrypted")
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
