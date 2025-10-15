/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package add

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/util"
)

// ProgressHandler manages progress display and user feedback for add operations
type ProgressHandler struct {
	Verbose bool
	Quiet   bool
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(verbose, quiet bool) *ProgressHandler {
	return &ProgressHandler{
		Verbose: verbose,
		Quiet:   quiet,
	}
}

// DisplayInitialProgress shows initial progress for batch operations
func (ph *ProgressHandler) DisplayInitialProgress(fileCount int) {
	if fileCount > 1 && !ph.Quiet {
		fmt.Printf("Starting batch processing of %d files...\n\n", fileCount)
	}
}

// DisplayFileProgress shows progress for individual file processing
func (ph *ProgressHandler) DisplayFileProgress(fileName string, index, total int, filePairs []FilePair) {
	if total > 1 && !ph.Quiet {
		fmt.Printf("[%d/%d] Processing: %s → %s\n",
			index+1, total, filepath.Base(filePairs[index].Source), filePairs[index].Destination)
	} else if !ph.Quiet {
		fmt.Printf("Processing: %s\n", filePairs[index].Source)
	}
}

// DisplayFileMetadata shows file metadata for confirmation
func (ph *ProgressHandler) DisplayFileMetadata(fileName string, sizeInBytes int64, modTime time.Time, tags []string) {
	if !ph.Quiet && (ph.Verbose) {
		fmt.Printf("  Size: %s (%d bytes)\n", util.HumanReadableSize(sizeInBytes), sizeInBytes)
		fmt.Printf("  Modified: %s\n", modTime.Format(time.RFC3339))
		if len(tags) > 0 {
			fmt.Printf("  Tags: %s\n", formatTags(tags))
		}
	}
}

// DisplayProcessingError shows processing errors
func (ph *ProgressHandler) DisplayProcessingError(fileName string, err error) {
	if !ph.Quiet {
		fmt.Printf("✗ %s: %v\n", filepath.Base(fileName), err)
	}
}

// DisplayProcessingSuccess shows successful processing
func (ph *ProgressHandler) DisplayProcessingSuccess(result ProcessResult, totalFiles int) {
	if !ph.Quiet {
		if totalFiles > 1 {
			fmt.Printf("✓ %s (%d chunks", result.FileName, result.ChunkCount)
			if result.SpaceSavings.SpaceSaved > 0 {
				fmt.Printf(", %s saved", util.HumanReadableSize(result.SpaceSavings.SpaceSaved))
			}
			fmt.Printf(")\n")
		} else {
			fmt.Printf("✓ File added to vault: %s\n", result.FileName)
			fmt.Printf("✓ %d chunks stored in vault\n", result.ChunkCount)
			if result.SpaceSavings.SpaceSaved > 0 {
				fmt.Printf("✓ Space saved: %s (%.1f%%)\n",
					util.HumanReadableSize(result.SpaceSavings.SpaceSaved),
					result.SpaceSavings.SpaceSavedPct)
			}
			fmt.Printf("✓ Manifest written to .sietch/manifests/%s.yaml\n", result.FileName)
		}
	}
}

// DisplaySymlinkResolution shows symlink resolution info
func (ph *ProgressHandler) DisplaySymlinkResolution(source, target string) {
	if ph.Verbose && !ph.Quiet {
		fmt.Printf("  Resolved symlink: %s → %s\n", source, target)
	}
}

// DisplayBatchSummary shows summary of batch processing
func (ph *ProgressHandler) DisplayBatchSummary(totalFiles, successCount int, failedFiles []string, totalSpaceSavings SpaceSavings, vaultConfig interface{}) {
	if ph.Quiet {
		return
	}

	fmt.Printf("\n=== Batch Processing Summary ===\n")
	fmt.Printf("Total files: %d\n", totalFiles)
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

	// Note: vaultConfig parameter would need proper typing based on actual config struct
	// For now, we'll skip the vault configuration display as it requires the actual config type
}

// formatTags formats tags for display
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return joinStrings(tags, ", ")
}

// joinStrings joins strings with a separator (simple implementation)
func joinStrings(strings []string, separator string) string {
	if len(strings) == 0 {
		return ""
	}
	if len(strings) == 1 {
		return strings[0]
	}

	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += separator + strings[i]
	}
	return result
}

// GetCommandFlags extracts common command flags
func GetCommandFlags(cmd *cobra.Command) (recursive, includeHidden, verbose, quiet bool, err error) {
	recursive, err = cmd.Flags().GetBool("recursive")
	if err != nil {
		return false, false, false, false, fmt.Errorf("error parsing recursive flag: %v", err)
	}

	includeHidden, err = cmd.Flags().GetBool("include-hidden")
	if err != nil {
		return false, false, false, false, fmt.Errorf("error parsing include-hidden flag: %v", err)
	}

	verbose, err = cmd.Flags().GetBool("verbose")
	if err != nil {
		return false, false, false, false, fmt.Errorf("error parsing verbose flag: %v", err)
	}

	quiet, err = cmd.Flags().GetBool("quiet")
	if err != nil {
		return false, false, false, false, fmt.Errorf("error parsing quiet flag: %v", err)
	}

	return recursive, includeHidden, verbose, quiet, nil
}
