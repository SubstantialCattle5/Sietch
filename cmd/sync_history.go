package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sort"

	"github.com/spf13/cobra"
)

type FileInfo struct {
    Path string `json:"path"`
    Size int64  `json:"size"` // in bytes
}

type SyncHistoryEntry struct {
	ID                string   	 `json:"id"`
	Timestamp         string   	 `json:"timestamp"`
	PeerID            string   	 `json:"peer_id"`
	PeerName          string   	 `json:"peer_name"`
	FilesTransferred  int      	 `json:"files_transferred"`
	ChunksTransferred int      	 `json:"chunks_transferred"`
	ChunksDeduped     int      	 `json:"chunks_deduplicated"`
	BytesTransferred  int64    	 `json:"bytes_transferred"`
	DurationMs        int64    	 `json:"duration_ms"`
	Status            string   	 `json:"status"`
	Files             []FileInfo `json:"files"`
	Error             string     `json:"error,omitempty"`
}

type SyncHistoryFile struct {
	Syncs []SyncHistoryEntry `json:"syncs"`
}

var (
	historyPeer   string
	historyFailed bool
	historyID     string
	historyExport bool
)

var syncHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View sync history",
	Long:  `View recent sync operations, filter by peer or failed syncs, export to CSV, or view detailed info.`,
	Run: func(cmd *cobra.Command, args []string) {
		showSyncHistory()
	},
}

func init() {
	rootCmd.AddCommand(syncHistoryCmd)
	syncHistoryCmd.Flags().StringVar(&historyPeer, "peer", "", "Filter by peer name")
	syncHistoryCmd.Flags().BoolVar(&historyFailed, "failed", false, "Show only failed syncs")
	syncHistoryCmd.Flags().StringVar(&historyID, "id", "", "Show detailed info for specific sync ID")
	syncHistoryCmd.Flags().BoolVar(&historyExport, "export", false, "Export sync history to CSV")
}

func ensureHistoryFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %v", err)
	}

	dir := filepath.Join(home, ".sietch")
	path := filepath.Join(dir, "sync-history.json")

	// Create folder if missing
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// Create file with empty JSON if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		empty := SyncHistoryFile{Syncs: []SyncHistoryEntry{}}
		data, _ := json.MarshalIndent(empty, "", "  ")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", fmt.Errorf("failed to create file: %v", err)
		}
	}

	return path, nil
}

func appendSyncHistory(entry SyncHistoryEntry) error {
    historyPath, err := ensureHistoryFile()
    if err != nil {
        return err
    }

    file, err := os.OpenFile(historyPath, os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    var history SyncHistoryFile
    if err := json.NewDecoder(file).Decode(&history); err != nil {
        return err
    }

    history.Syncs = append(history.Syncs, entry)

    // reset file to start
    file.Seek(0, 0)
    file.Truncate(0)

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(history)
}

func showSyncHistory() {
	// locate sync-history.json
	historyPath, err := ensureHistoryFile()
	if err != nil {
		fmt.Println(err)
		return
	}

	file, err := os.Open(historyPath)
	if err != nil {
		fmt.Println("Cannot open sync history file:", err)
		return
	}
	defer file.Close()

	var history SyncHistoryFile
	if err := json.NewDecoder(file).Decode(&history); err != nil {
		fmt.Println("Failed to decode history JSON:", err)
		return
	}

	entries := history.Syncs

	// filter by peer
	if historyPeer != "" {
		filtered := []SyncHistoryEntry{}
		for _, e := range entries {
			if e.PeerName == historyPeer {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// filter failed only
	if historyFailed {
		filtered := []SyncHistoryEntry{}
		for _, e := range entries {
			if e.Status == "failed" {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// show detailed info if --id is provided
	if historyID != "" {
		for _, e := range entries {
			if e.ID == historyID {
				showDetailedEntry(e)
				return
			}
		}
		fmt.Println("No sync found with ID:", historyID)
		return
	}

	// if --export, write CSV
	if historyExport {
		exportCSV(entries)
		return
	}

	// sort entries by timestamp (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	// otherwise list last 10 entries
	count := 10
	if len(entries) < 10 {
		count = len(entries)
	}
	fmt.Println("Last", count, "syncs:")
	for i := len(entries) - count; i < len(entries); i++ {
		e := entries[i]
		status := "✅"
		if e.Status == "failed" {
			status = "❌ Failed: " + e.Error
		} else if e.FilesTransferred == 0 {
			status = "⚠️ No changes"
		}
		fmt.Printf("  %s  %s  %d files  %.2f MB  %s\n",
			e.Timestamp[:16], e.PeerName, e.FilesTransferred, float64(e.BytesTransferred)/1024/1024, status)
	}
}

func showDetailedEntry(e SyncHistoryEntry) {
	fmt.Printf("Sync ID: %s\nDate: %s\nPeer: %s (%s)\nDirection: bidirectional\n",
		e.ID, e.Timestamp, e.PeerName, e.PeerID)
	fmt.Println("Files transferred:")
	for _, f := range e.Files {
		fmt.Printf("  - %s (%.2f MB)\n", f.Path, float64(f.Size)/1024/1024)
	}
	fmt.Printf("Chunks: %d transferred, %d deduplicated\n", e.ChunksTransferred, e.ChunksDeduped)
	fmt.Printf("Duration: %.2fs\n", float64(e.DurationMs)/1000)
	fmt.Printf("Status: %s\n", e.Status)
	if e.Error != "" {
		fmt.Println("Error:", e.Error)
	}
}

func exportCSV(entries []SyncHistoryEntry) {
	home, _ := os.UserHomeDir()
	outPath := filepath.Join(home, ".sietch", "sync-history.csv")
	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Println("Cannot create CSV file:", err)
		return
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	writer.Write([]string{"id","timestamp","peer_id","peer_name","files_transferred","chunks_transferred","chunks_deduplicated","bytes_transferred","duration_ms","status","files","error"})
	for _, e := range entries {
		writer.Write([]string{
			e.ID,
			e.Timestamp,
			e.PeerID,
			e.PeerName,
			strconv.Itoa(e.FilesTransferred),
			strconv.Itoa(e.ChunksTransferred),
			strconv.Itoa(e.ChunksDeduped),
			strconv.FormatInt(e.BytesTransferred, 10),
			strconv.FormatInt(e.DurationMs, 10),
			e.Status,
			fmt.Sprintf("%v", e.Files),
			e.Error,
		})
	}
	fmt.Println("Exported CSV to", outPath)
}
