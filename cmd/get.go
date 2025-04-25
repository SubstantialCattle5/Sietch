/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/util"
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
		force, _ := cmd.Flags().GetBool("force")
		skipEncryption, _ := cmd.Flags().GetBool("skip-decryption")

		fmt.Printf("Retrieving %s from vault\n", filePath)

		// Load the manifest for the requested file
		manifestName := filepath.Base(filePath)
		fileManifest, err := manifest.LoadFileManifest(vaultRoot, manifestName)
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
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %v", err)
		}

		// Create output file
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer outputFile.Close()

		// Handle decryption based on vault configuration
		var passphrase string
		if !skipEncryption && vaultConfig.Encryption.Type == "aes" && vaultConfig.Encryption.PassphraseProtected {
			// Prompt for passphrase if the key is protected
			passphrasePrompt := promptui.Prompt{
				Label: "Enter encryption passphrase",
				Mask:  '*',
			}
			passphrase, err = passphrasePrompt.Run()
			if err != nil {
				return fmt.Errorf("failed to get passphrase: %v", err)
			}
		}
		fmt.Print(passphrase)

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
			if !skipEncryption && vaultConfig.Encryption.Type == "aes" {
				if len(chunkData) == 0 {
					return fmt.Errorf("chunk %s is empty", chunkHash)
				}

				// Decrypt the data
				decryptedData, err := encryption.AesDecryption(
					string(chunkData),
					vaultRoot,
				)
				if err != nil {
					return fmt.Errorf("failed to decrypt chunk %s: %v", chunkHash, err)
				}

				// Verify chunk integrity by hashing the decrypted content
				if !skipEncryption && chunkRef.Hash != "" {
					// You would implement hash verification here
					// Example: if util.SHA256Sum([]byte(decryptedData)) != chunkRef.Hash {
					//    return fmt.Errorf("chunk integrity check failed")
					// }
				}

				chunkData = []byte(decryptedData)
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
	getCmd.Flags().BoolP("force", "f", false, "Force overwrite if file exists at destination")
	getCmd.Flags().Bool("skip-decryption", false, "Skip decryption and retrieve raw chunks (for recovery)")
}
