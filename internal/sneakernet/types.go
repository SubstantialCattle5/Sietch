package sneakernet

import (
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
)

// VaultInfo represents discovered vault metadata
type VaultInfo struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	VaultID    string    `json:"vault_id"`
	FileCount  int       `json:"file_count"`
	TotalSize  int64     `json:"total_size"`
	LastAccess time.Time `json:"last_access"`
	CreatedAt  time.Time `json:"created_at"`
}

// SneakAnalysis represents transfer analysis results
type SneakAnalysis struct {
	NewFiles        []config.FileManifest `json:"new_files"`
	NewChunks       []string              `json:"new_chunks"`
	DuplicateChunks []string              `json:"duplicate_chunks"`
	Conflicts       []FileConflict        `json:"conflicts"`
	TotalSize       int64                 `json:"total_size"`
	TransferSize    int64                 `json:"transfer_size"`
	DuplicateSize   int64                 `json:"duplicate_size"`
}

// FileConflict represents a file naming conflict
type FileConflict struct {
	FilePath   string              `json:"file_path"`
	SourceInfo config.FileManifest `json:"source_info"`
	DestInfo   config.FileManifest `json:"dest_info"`
	Resolution ConflictResolution  `json:"resolution"`
}

// ConflictResolution represents how to handle conflicts
type ConflictResolution struct {
	Action  string `json:"action"`   // "skip", "overwrite", "rename"
	NewName string `json:"new_name"` // for rename action
}

// TransferResult contains the results of a sneakernet transfer
type TransferResult struct {
	FilesTransferred  int            `json:"files_transferred"`
	ChunksTransferred int            `json:"chunks_transferred"`
	ChunksSkipped     int            `json:"chunks_skipped"`
	BytesTransferred  int64          `json:"bytes_transferred"`
	Duration          time.Duration  `json:"duration"`
	Conflicts         []FileConflict `json:"conflicts"`
	Errors            []string       `json:"errors"`
}

// SneakTransfer handles sneakernet transfer operations
type SneakTransfer struct {
	SourceVault     string   `json:"source_vault"`
	DestVault       string   `json:"dest_vault"`
	FilePatterns    []string `json:"file_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	AutoResolve     bool     `json:"auto_resolve"`
	DryRun          bool     `json:"dry_run"`
	Verbose         bool     `json:"verbose"`
}
