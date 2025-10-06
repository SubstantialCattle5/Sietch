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

		// Filter and sort files
		files := filterAndSortFiles(manifest.Files, filterPath, sortBy)

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
			displayLongFormat(files, showTags)
		} else {
			displayShortFormat(files, showTags)
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
func displayLongFormat(files []config.FileManifest, showTags bool) {
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
	}
}

// Display files in short format
func displayShortFormat(files []config.FileManifest, showTags bool) {
	for _, file := range files {
		path := file.Destination + file.FilePath
		if showTags && len(file.Tags) > 0 {
			tags := strings.Join(file.Tags, ", ")
			fmt.Printf("%s [%s]\n", path, tags)
		} else {
			fmt.Println(path)
		}
	}
}

func init() {
	rootCmd.AddCommand(lsCmd)

	// Add flags
	lsCmd.Flags().BoolP("long", "l", false, "Use long listing format")
	lsCmd.Flags().BoolP("tags", "t", false, "Show file tags")
	lsCmd.Flags().StringP("sort", "s", "path", "Sort by: name, size, time, path")
}
