package history

import (
	"encoding/json"
	"os"
)

// SyncRecord represents a single sync operation
type SyncRecord struct {
	ID                 string   `json:"id"`
	Timestamp          string   `json:"timestamp"`
	PeerID             string   `json:"peer_id"`
	PeerName           string   `json:"peer_name"`
	FilesTransferred   int      `json:"files_transferred"`
	ChunksTransferred  int      `json:"chunks_transferred"`
	ChunksDeduplicated int      `json:"chunks_deduplicated"`
	BytesTransferred   int64    `json:"bytes_transferred"`
	DurationMs         int64    `json:"duration_ms"`
	Status             string   `json:"status"`
	Files              []string `json:"files"`
	Error              string   `json:"error"`
}

// History wraps multiple SyncRecords
type History struct {
	Syncs []SyncRecord `json:"syncs"`
}

// AddRecord appends a new record to the JSON file
func AddRecord(path string, record SyncRecord) error {
	h, _ := LoadHistory(path)       // ignore errors, create new if missing
	h.Syncs = append(h.Syncs, record)

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadHistory loads history from a JSON file
func LoadHistory(path string) (*History, error) {
	h := &History{}

	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist, return empty history
		if os.IsNotExist(err) {
			return h, nil
		}
		return nil, err
	}

	err = json.Unmarshal(data, h)
	if err != nil {
		return nil, err
	}

	return h, nil
}
