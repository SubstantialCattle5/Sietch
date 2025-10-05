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
	"github.com/substantialcattle5/sietch/internal/fs"
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
			displayShortFormat(files, showTags, showDedup, chunkRefs)
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
			sharedChunks, savedBytes, sharedWith := computeDedupStatsForFile(file, chunkRefs)
			// Format saved size
			savedStr := util.HumanReadableSize(int64(savedBytes))
			// Format shared_with string with truncation
			sharedWithStr := formatSharedWith(sharedWith, 10)
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

// Display files in short format
func displayShortFormat(files []config.FileManifest, showTags, showDedup bool, chunkRefs map[string][]string) {
	for _, file := range files {
		path := file.Destination + file.FilePath
		if showTags && len(file.Tags) > 0 {
			tags := strings.Join(file.Tags, ", ")
			fmt.Printf("%s [%s]\n", path, tags)
		} else {
			fmt.Println(path)
		}

		// Dedup stats line if requested
		if showDedup && chunkRefs != nil {
			sharedChunks, savedBytes, sharedWith := computeDedupStatsForFile(file, chunkRefs)
			savedStr := util.HumanReadableSize(int64(savedBytes))
			sharedWithStr := formatSharedWith(sharedWith, 10)
			if len(sharedWith) == 0 {
				fmt.Printf("  shared_chunks: %d  saved: %s\n", sharedChunks, savedStr)
			} else {
				fmt.Printf("  shared_chunks: %d  saved: %s  shared_with: %s\n", sharedChunks, savedStr, sharedWithStr)
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

// computeDedupStatsForFile calculates dedup stats by consulting chunkRefs map.
// Uses EncryptedSize if present, otherwise Size, otherwise falls back to default chunk size.
func computeDedupStatsForFile(file config.FileManifest, chunkRefs map[string][]string) (sharedChunks int, savedBytes int64, sharedWith []string) {
	// Default chunk size assumption (matches docs): 4 MiB
	const defaultChunkSize int64 = 4 * 1024 * 1024

	sharedWithSet := make(map[string]struct{})
	filePath := file.Destination + file.FilePath

	for _, c := range file.Chunks {
		chunkID := c.Hash
		if chunkID == "" {
			chunkID = c.EncryptedHash
		}
		if chunkID == "" {
			continue
		}

		refs, ok := chunkRefs[chunkID]
		if !ok {
			continue
		}
		if len(refs) > 1 {
			sharedChunks++

			// Prefer encrypted size if available (actual stored size), fallback to plaintext size
			var chunkSize int64
			if c.EncryptedSize > 0 {
				chunkSize = c.EncryptedSize
			} else if c.Size > 0 {
				chunkSize = c.Size
			} else {
				chunkSize = defaultChunkSize
			}
			savedBytes += chunkSize

			for _, other := range refs {
				if other == filePath {
					continue
				}
				sharedWithSet[other] = struct{}{}
			}
		}
	}

	sharedWith = make([]string, 0, len(sharedWithSet))
	for s := range sharedWithSet {
		sharedWith = append(sharedWith, s)
	}
	// sort for deterministic output
	sort.Strings(sharedWith)
	return
}

// formatSharedWith joins sharedWith and truncates after limit entries, showing (+N more)
func formatSharedWith(list []string, limit int) string {
	if len(list) == 0 {
		return ""
	}
	if len(list) <= limit {
		return strings.Join(list, ", ")
	}
	visible := list[:limit]
	return fmt.Sprintf("%s (+%d more)", strings.Join(visible, ", "), len(list)-limit)
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
