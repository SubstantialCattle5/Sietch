/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/add"
	"github.com/substantialcattle5/sietch/internal/progress"
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
		filePairs, err := add.ParseFileArguments(args)
		if err != nil {
			return err
		}

		// Get command flags
		recursive, includeHidden, verbose, quiet, err := add.GetCommandFlags(cmd)
		if err != nil {
			return err
		}

		// Expand directories if needed
		filePairs, err = add.ExpandDirectories(filePairs, recursive, includeHidden)
		if err != nil {
			return err
		}

		// Get tags from command
		tags, err := add.GetTagsFromCommand(cmd)
		if err != nil {
			return err
		}

		// Setup vault context
		vaultCtx, err := add.SetupVaultContext(cmd)
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

		// Create progress handler
		progressHandler := add.NewProgressHandler(verbose, quiet)

		// Process each file pair
		successCount := 0
		var failedFiles []string
		var processResults []add.ProcessResult

		// Show initial progress for multiple files
		progressHandler.DisplayInitialProgress(len(filePairs))

		for i, pair := range filePairs {
			// Display file progress
			progressHandler.DisplayFileProgress(pair.Source, i, len(filePairs), filePairs)

			// Process the file using core logic
			result := add.ProcessFile(ctx, pair, vaultCtx.ChunkSize, vaultCtx.VaultRoot, vaultCtx.Passphrase, progressMgr, tags)
			processResults = append(processResults, result)

			if result.Success {
				successCount++
				progressHandler.DisplayProcessingSuccess(result, len(filePairs))
			} else {
				errorMsg := fmt.Sprintf("✗ %s: %v", result.FileName, result.Error)
				progressHandler.DisplayProcessingError(result.FileName, result.Error)
				failedFiles = append(failedFiles, errorMsg)
			}
		}

		// Cleanup progress manager
		progressMgr.Cleanup()

		// Calculate total space savings
		totalSpaceSavings := add.CalculateTotalSpaceSavings(processResults)

		// Display batch summary
		progressHandler.DisplayBatchSummary(len(filePairs), successCount, failedFiles, totalSpaceSavings, vaultCtx.VaultConfig)

		// Show vault configuration details if successful
		if successCount > 0 && !quiet {
			fmt.Printf("\n📋 Vault Configuration:\n")
			fmt.Printf("  • Encryption: %s", vaultCtx.VaultConfig.Encryption.Type)
			if vaultCtx.VaultConfig.Encryption.PassphraseProtected {
				fmt.Printf(" (passphrase protected)")
			}
			fmt.Println()

			fmt.Printf("  • Compression: %s\n", vaultCtx.VaultConfig.Compression)
			fmt.Printf("  • Chunking: %s (size: %s)\n", vaultCtx.VaultConfig.Chunking.Strategy, vaultCtx.VaultConfig.Chunking.ChunkSize)

			// Show total space savings if compression is used
			if vaultCtx.VaultConfig.Compression != "none" && totalSpaceSavings.SpaceSaved > 0 {
				fmt.Printf("\n💾 Total Space Savings:\n")
				fmt.Printf("  • Original size: %s\n", util.HumanReadableSize(totalSpaceSavings.OriginalSize))
				fmt.Printf("  • Compressed size: %s\n", util.HumanReadableSize(totalSpaceSavings.CompressedSize))
				fmt.Printf("  • Space saved: %s (%.1f%%)\n",
					util.HumanReadableSize(totalSpaceSavings.SpaceSaved),
					totalSpaceSavings.SpaceSavedPct)
			}
		}

		// Return error only if all files failed
		if successCount == 0 {
			return fmt.Errorf("all files failed to process")
		}

		return nil
	},
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
