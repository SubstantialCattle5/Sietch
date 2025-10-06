/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/internal/fs"
	lsui "github.com/substantialcattle5/sietch/internal/ls"
	"github.com/substantialcattle5/sietch/util"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List files in the Sietch vault",
	Long: `List files stored in your Sietch vault.

This command displays information about files stored in your vault.
By default, it shows files at the vault root, but you can specify a
path within the vault to list files in that directory.

Examples:
  sietch ls              # List all files in the vault
  sietch ls docs/        # List files in the docs directory
  sietch ls --long       # Show detailed file information
  sietch ls --tags       # Show file tags
  sietch ls --sort=size  # Sort files by size`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Get filter path
		filterPath := ""
		if len(args) > 0 {
			filterPath = args[0]
		}

		// Find vault root
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Create a vault manager
		manager, err := config.NewManager(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to create vault manager: %v", err)
		}

		// Get the vault manifest
		manifest, err := manager.GetManifest()
		if err != nil {
			return fmt.Errorf("failed to get vault manifest: %v", err)
		}

		// Get display options
		long, _ := cmd.Flags().GetBool("long")
		showTags, _ := cmd.Flags().GetBool("tags")
		sortBy, _ := cmd.Flags().GetString("sort")
		showDedup, _ := cmd.Flags().GetBool("dedup-stats")

		// Filter and sort files
		files := filterAndSortFiles(manifest.Files, filterPath, sortBy)

		// Build chunk -> files index only if dedup stats requested
		var chunkRefs map[string][]string
		if showDedup {
			chunkRefs = buildChunkIndex(manifest.Files)
		}

		// Display the files
		if len(files) == 0 {
			if filterPath != "" {
				fmt.Printf("No files found in '%s'\n", filterPath)
			} else {
				fmt.Println("No files found in vault")
			}
			return nil
		}

		if long {
			displayLongFormat(files, showTags, showDedup, chunkRefs)
		} else {
			lsui.DisplayShortFormat(files, showTags, showDedup, chunkRefs)
		}

		return nil
	},
}

// Filter files by path and sort them according to the specified criteria
func filterAndSortFiles(files []config.FileManifest, filterPath, sortBy string) []config.FileManifest {
	// Filter files
	var filtered []config.FileManifest
	for _, file := range files {
		if filterPath == "" || strings.HasPrefix(file.Destination, filterPath) {
			filtered = append(filtered, file)
		}
	}

	// Sort files
	switch strings.ToLower(sortBy) {
	case "name":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].FilePath < filtered[j].FilePath
		})
	case "size":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Size > filtered[j].Size
		})
	case "time":
		sort.Slice(filtered, func(i, j int) bool {
			timeI, _ := time.Parse(time.RFC3339, filtered[i].ModTime)
			timeJ, _ := time.Parse(time.RFC3339, filtered[j].ModTime)
			return timeI.After(timeJ)
		})
	default:
		// Default sort by path
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Destination < filtered[j].Destination
		})
	}

	return filtered
}

// Display files in long format with detailed information
// showDedup = whether to include dedup stats; chunkRefs is map[chunkID][]filePaths
func displayLongFormat(files []config.FileManifest, showTags, showDedup bool, chunkRefs map[string][]string) {
	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	if showTags {
		fmt.Fprintln(w, "SIZE\tMODIFIED\tCHUNKS\tPATH\tTAGS")
	} else {
		fmt.Fprintln(w, "SIZE\tMODIFIED\tCHUNKS\tPATH")
	}

	// Print each file
	for _, file := range files {
		// Parse and format time
		modTime, _ := time.Parse(time.RFC3339, file.ModTime)
		timeFormat := modTime.Format("2006-01-02 15:04:05")

		// Format output
		if showTags {
			tags := strings.Join(file.Tags, ", ")
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				util.HumanReadableSize(file.Size),
				timeFormat,
				len(file.Chunks),
				file.Destination+file.FilePath,
				tags)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
				util.HumanReadableSize(file.Size),
				timeFormat,
				len(file.Chunks),
				file.Destination+file.FilePath)
		}

		// Dedup stats (print an indented stats line after the file line)
		if showDedup && chunkRefs != nil {
			sharedChunks, savedBytes, sharedWith := deduplication.ComputeDedupStatsForFile(file, chunkRefs)
			// Format saved size
			savedStr := util.HumanReadableSize(savedBytes)
			// Format shared_with string with truncation
			sharedWithStr := lsui.FormatSharedWith(sharedWith, 10)
			// Print as indented info (not part of the tabwriter)
			if len(sharedWith) == 0 {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "", "", "", "") // ensure tabwriter alignment
				fmt.Fprintf(w, "    shared_chunks: %d\t saved: %s\n", sharedChunks, savedStr)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "", "", "", "") // alignment spacer
				fmt.Fprintf(w, "    shared_chunks: %d\t saved: %s\t shared_with: %s\n", sharedChunks, savedStr, sharedWithStr)
			}
		}
	}
}

// buildChunkIndex creates a mapping chunkID -> []filePaths using the manifest file list.
// Uses ChunkRef.Hash as the chunk identifier.
func buildChunkIndex(files []config.FileManifest) map[string][]string {
	chunkRefs := make(map[string][]string)
	for _, f := range files {
		fp := f.Destination + f.FilePath
		for _, c := range f.Chunks {
			// use the Hash field as the chunk identifier
			chunkID := c.Hash
			if chunkID == "" {
				// fallback: if Hash is empty, use EncryptedHash
				chunkID = c.EncryptedHash
			}
			if chunkID == "" {
				// skip weird entries
				continue
			}
			chunkRefs[chunkID] = append(chunkRefs[chunkID], fp)
		}
	}
	return chunkRefs
}

func init() {
	rootCmd.AddCommand(lsCmd)

	// Add flags
	lsCmd.Flags().BoolP("long", "l", false, "Use long listing format")
	lsCmd.Flags().BoolP("tags", "t", false, "Show file tags")
	lsCmd.Flags().StringP("sort", "s", "path", "Sort by: name, size, time, path")

	// New dedup-stats flag
	lsCmd.Flags().BoolP("dedup-stats", "d", false, "Show per-file deduplication statistics")
}
