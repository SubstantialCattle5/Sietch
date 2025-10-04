package p2p

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"

    "github.com/google/uuid"
)

type SyncRecord struct {
    ID                 string   `json:"id"`
    Timestamp          string   `json:"timestamp"` // ISO8601
    PeerID             string   `json:"peer_id"`
    PeerName           string   `json:"peer_name"`
    FilesTransferred   int      `json:"files_transferred"`
    ChunksTransferred  int      `json:"chunks_transferred"`
    ChunksDeduplicated int      `json:"chunks_deduplicated"`
    BytesTransferred   int64    `json:"bytes_transferred"`
    DurationMS         int64    `json:"duration_ms"`
    Status             string   `json:"status"` // "success", "failed", "no_changes"
    Files              []string `json:"files"`
    Error              string   `json:"error,omitempty"`
}

type SyncHistory struct {
    Syncs []SyncRecord `json:"syncs"`
}

// Returns the path to sync-history.json inside the vault
func getHistoryFilePath(vaultPath string) string {
    return filepath.Join(vaultPath, ".sietch", "sync-history.json")
}

// Load history from disk
func LoadSyncHistory(vaultPath string) (*SyncHistory, error) {
    path := getHistoryFilePath(vaultPath)
    history := &SyncHistory{}

    f, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            return history, nil // no history yet
        }
        return nil, err
    }
    defer f.Close()

    err = json.NewDecoder(f).Decode(history)
    if err != nil {
        return nil, err
    }
    return history, nil
}

// Save history to disk
func SaveSyncHistory(vaultPath string, history *SyncHistory) error {
    path := getHistoryFilePath(vaultPath)

    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ")
    return encoder.Encode(history)
}

// Add a new record
func AddSyncRecord(vaultPath string, record SyncRecord) error {
    history, err := LoadSyncHistory(vaultPath)
    if err != nil {
        return err
    }

    history.Syncs = append([]SyncRecord{record}, history.Syncs...) // newest first

    return SaveSyncHistory(vaultPath, history)
}

// Helper to create a new SyncRecord
func NewSyncRecord(peerID, peerName string, files []string, chunksTransferred, chunksDedup int, bytes int64, duration time.Duration, status string, errMsg string) SyncRecord {
    return SyncRecord{
        ID:                 uuid.New().String(),
        Timestamp:          time.Now().UTC().Format(time.RFC3339),
        PeerID:             peerID,
        PeerName:           peerName,
        FilesTransferred:   len(files),
        ChunksTransferred:  chunksTransferred,
        ChunksDeduplicated: chunksDedup,
        BytesTransferred:   bytes,
        DurationMS:         int64(duration / time.Millisecond),
        Status:             status,
        Files:              files,
        Error:              errMsg,
    }
}

// SyncHistoryEntry is an in-memory detailed log used during sync operations.
type SyncHistoryEntry struct {
	ID                string     `json:"id"`
	Timestamp         string     `json:"timestamp"`
	PeerID            string     `json:"peer_id"`
	PeerName          string     `json:"peer_name"`
	Status            string     `json:"status"`
	Error             string     `json:"error,omitempty"`
	FilesTransferred  int        `json:"files_transferred,omitempty"`
	ChunksTransferred int        `json:"chunks_transferred,omitempty"`
	ChunksDeduped     int        `json:"chunks_deduped,omitempty"`
	BytesTransferred  int64      `json:"bytes_transferred,omitempty"`
	DurationMs        int64      `json:"duration_ms,omitempty"`
	Files             []FileInfo `json:"files,omitempty"`
}

// FileInfo stores per-file transfer details for sync logs.
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// appendSyncHistory bridges between SyncHistoryEntry (detailed)
// and SyncRecord (persistent JSON storage)
func appendSyncHistory(entry SyncHistoryEntry) error {
	vaultPath := "." // TODO: if available, inject real vault path
	files := make([]string, len(entry.Files))
	for i, f := range entry.Files {
		files[i] = f.Path
	}

	record := SyncRecord{
		ID:                 entry.ID,
		Timestamp:          entry.Timestamp,
		PeerID:             entry.PeerID,
		PeerName:           entry.PeerName,
		FilesTransferred:   entry.FilesTransferred,
		ChunksTransferred:  entry.ChunksTransferred,
		ChunksDeduplicated: entry.ChunksDeduped,
		BytesTransferred:   entry.BytesTransferred,
		DurationMS:         entry.DurationMs,
		Status:             entry.Status,
		Files:              files,
		Error:              entry.Error,
	}

	return AddSyncRecord(vaultPath, record)
}
