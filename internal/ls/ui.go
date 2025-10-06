
package ls

import (
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/util"
)

// FormatSharedWith joins sharedWith and truncates after limit entries, showing (+N more)
func FormatSharedWith(list []string, limit int) string {
	if len(list) == 0 {
		return ""
	}
	if len(list) <= limit {
		return strings.Join(list, ", ")
	}
	visible := list[:limit]
	return fmt.Sprintf("%s (+%d more)", strings.Join(visible, ", "), len(list)-limit)
}

// DisplayShortFormat prints the short listing for files and optionally shows dedup stats.
// This mirrors the previous displayShortFormat that lived in cmd/ls.go.
func DisplayShortFormat(files []config.FileManifest, showTags, showDedup bool, chunkRefs map[string][]string) {
	for _, file := range files {
		path := file.Destination + file.FilePath
		if showTags && len(file.Tags) > 0 {
			tags := strings.Join(file.Tags, ", ")
			// Print file and tags
			// note: fmt.Printf used intentionally to match previous behavior
			// so command output layout remains identical
			// (no tabwriter here)
			fmt.Printf("%s [%s]\n", path, tags)
		} else {
			fmt.Println(path)
		}

		// Dedup stats line if requested
		if showDedup && chunkRefs != nil {
			sharedChunks, savedBytes, sharedWith := deduplication.ComputeDedupStatsForFile(file, chunkRefs)
			savedStr := util.HumanReadableSize(savedBytes)
			sharedWithStr := FormatSharedWith(sharedWith, 10)
			if len(sharedWith) == 0 {
				fmt.Printf("  shared_chunks: %d  saved: %s\n", sharedChunks, savedStr)
			} else {
				fmt.Printf("  shared_chunks: %d  saved: %s  shared_with: %s\n", sharedChunks, savedStr, sharedWithStr)
			}
		}
	}
}

