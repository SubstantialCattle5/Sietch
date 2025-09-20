/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/sneakernet"
	"github.com/substantialcattle5/sietch/util"
)

// sneakCmd represents the sneakernet command
var sneakCmd = &cobra.Command{
	Use:   "sneak [flags]",
	Short: "Transfer vault data via sneakernet (USB, physical media)",
	Long: `Transfer files and chunks from another vault via sneakernet (USB drives, external media).

This command discovers vaults on mounted drives and merges their data into your current vault
without overwriting existing data. Perfect for offline data transfer scenarios.

Examples:
  sietch sneak                                    # Interactive mode - discover and select vaults
  sietch sneak --source /media/usb/research-vault # Transfer from specific vault
  sietch sneak --source /backup/vault --dry-run   # Show what would be transferred
  sietch sneak --files "*.pdf,docs/*" --source /usb/vault # Transfer specific file patterns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		sourcePath, _ := cmd.Flags().GetString("source")
		destPath, _ := cmd.Flags().GetString("dest")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		filePatterns, _ := cmd.Flags().GetStringSlice("files")
		excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
		autoResolve, _ := cmd.Flags().GetBool("auto-resolve")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Find destination vault (current vault if not specified)
		if destPath == "" {
			var err error
			destPath, err = fs.FindVaultRoot()
			if err != nil {
				return fmt.Errorf("not inside a vault and no destination specified: %v", err)
			}
		}

		// Validate destination vault exists
		if !sneakernet.IsValidVault(destPath) {
			return fmt.Errorf("destination is not a valid vault: %s", destPath)
		}

		fmt.Printf("🎯 Destination vault: %s\n", destPath)

		// Source vault discovery or validation
		var sourceVault string
		if !sneakernet.IsValidVault(sourcePath) {
			return fmt.Errorf("source is not a valid vault: %s", sourcePath)
		}

		// Create sneakernet transfer
		transfer := &sneakernet.SneakTransfer{
			SourceVault:     sourceVault,
			DestVault:       destPath,
			FilePatterns:    filePatterns,
			ExcludePatterns: excludePatterns,
			AutoResolve:     autoResolve,
			DryRun:          dryRun,
			Verbose:         verbose,
		}

		// Analyze the transfer
		fmt.Println("🔍 Analyzing vaults...")
		analysis, err := transfer.Analyze()
		if err != nil {
			return fmt.Errorf("analysis failed: %v", err)
		}

		// Display analysis results
		displayAnalysis(analysis)

		// Check if there's anything to transfer
		if len(analysis.NewFiles) == 0 && len(analysis.NewChunks) == 0 {
			fmt.Println("✅ Nothing to transfer - vaults are already in sync!")
			return nil
		}

		// Handle conflicts if any
		if len(analysis.Conflicts) > 0 && !autoResolve {
			fmt.Printf("\n⚠️  Found %d file conflicts that need resolution:\n", len(analysis.Conflicts))
			err := resolveConflictsInteractively(analysis.Conflicts)
			if err != nil {
				return fmt.Errorf("conflict resolution failed: %v", err)
			}
		}

		// Confirm transfer (unless dry-run)
		if !dryRun {
			if !confirmTransfer(analysis) {
				fmt.Println("Transfer cancelled.")
				return nil
			}

			// Execute the transfer
			fmt.Println("\n🚀 Starting sneakernet transfer...")
			result, err := transfer.Execute()
			if err != nil {
				return fmt.Errorf("transfer failed: %v", err)
			}

			// Display results
			displayTransferResults(result)
		} else {
			fmt.Println("\n🧪 Dry run completed - no actual transfer performed.")
		}

		return nil
	},
}

// displayAnalysis shows the transfer analysis results
func displayAnalysis(analysis *sneakernet.SneakAnalysis) {
	fmt.Println("\n📊 Sneakernet Analysis:")
	fmt.Println("======================")
	fmt.Printf("New files:        %d files\n", len(analysis.NewFiles))
	fmt.Printf("New chunks:       %d chunks (%s)\n", len(analysis.NewChunks), util.HumanReadableSize(analysis.TransferSize))
	fmt.Printf("Duplicate chunks: %d chunks (%s - will skip)\n", len(analysis.DuplicateChunks), util.HumanReadableSize(analysis.DuplicateSize))

	if len(analysis.Conflicts) > 0 {
		fmt.Printf("Conflicts:        %d files (need resolution)\n", len(analysis.Conflicts))
	}

	fmt.Printf("\nTotal transfer:   %s\n", util.HumanReadableSize(analysis.TransferSize))
}

// resolveConflictsInteractively handles file conflicts
func resolveConflictsInteractively(conflicts []sneakernet.FileConflict) error {
	for i := range conflicts {
		conflict := &conflicts[i]
		fmt.Printf("\n⚠️  Conflict %d of %d:\n", i+1, len(conflicts))
		fmt.Printf("File: %s\n", conflict.FilePath)
		fmt.Printf("Source:  Modified %s, Size: %s\n",
			conflict.SourceInfo.ModTime,
			util.HumanReadableSize(conflict.SourceInfo.Size))
		fmt.Printf("Current: Modified %s, Size: %s\n",
			conflict.DestInfo.ModTime,
			util.HumanReadableSize(conflict.DestInfo.Size))

		for {
			fmt.Print("\nChoose action [s]kip/[o]verwrite/[r]ename: ")
			var choice string
			_, err := fmt.Scanln(&choice)
			if err != nil {
				continue
			}

			switch strings.ToLower(choice) {
			case "s", "skip":
				conflict.Resolution.Action = "skip"
				fmt.Printf("✅ Will skip %s (keep current version)\n", conflict.FilePath)
				goto nextConflict
			case "o", "overwrite":
				conflict.Resolution.Action = "overwrite"
				fmt.Printf("✅ Will overwrite %s with source version\n", conflict.FilePath)
				goto nextConflict
			case "r", "rename":
				fmt.Print("Enter new name for source file: ")
				var newName string
				_, err := fmt.Scanln(&newName)
				if err != nil {
					fmt.Println("Invalid name, please try again.")
					continue
				}
				conflict.Resolution.Action = "rename"
				conflict.Resolution.NewName = newName
				fmt.Printf("✅ Will save source as %s\n", newName)
				goto nextConflict
			default:
				fmt.Println("Invalid choice. Please enter 's', 'o', or 'r'.")
			}
		}
	nextConflict:
	}

	return nil
}

// confirmTransfer asks user to confirm the transfer
func confirmTransfer(analysis *sneakernet.SneakAnalysis) bool {
	fmt.Printf("\n🤔 Ready to transfer %s in %d chunks. Continue? [y/N]: ",
		util.HumanReadableSize(analysis.TransferSize),
		len(analysis.NewChunks))

	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// displayTransferResults shows the final transfer results
func displayTransferResults(result *sneakernet.TransferResult) {
	fmt.Println("\n✅ Sneakernet transfer complete!")
	fmt.Printf("   Files transferred:    %d\n", result.FilesTransferred)
	fmt.Printf("   Chunks transferred:   %d\n", result.ChunksTransferred)
	fmt.Printf("   Chunks skipped:       %d (already present)\n", result.ChunksSkipped)
	fmt.Printf("   Data transferred:     %s\n", util.HumanReadableSize(result.BytesTransferred))
	fmt.Printf("   Duration:             %s\n", result.Duration.Round(time.Millisecond))

	if len(result.Conflicts) > 0 {
		fmt.Printf("   Conflicts resolved:   %d\n", len(result.Conflicts))
	}
}

func init() {
	rootCmd.AddCommand(sneakCmd)

	// Add command flags
	sneakCmd.Flags().StringP("source", "s", "", "Source vault path")
	sneakCmd.Flags().StringP("dest", "d", "", "Destination vault path (default: current vault)")
	sneakCmd.Flags().BoolP("dry-run", "n", false, "Show what would be transferred without doing it")
	sneakCmd.Flags().StringSliceP("files", "f", []string{}, "Specific file patterns to transfer")
	sneakCmd.Flags().StringSliceP("exclude", "e", []string{}, "File patterns to exclude")
	sneakCmd.Flags().BoolP("auto-resolve", "a", false, "Automatically resolve conflicts by renaming")
	sneakCmd.Flags().BoolP("interactive", "i", true, "Interactive mode for vault selection")
	sneakCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}
