/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package add

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/internal/progress"
)

// FilePair represents a source file and its destination path
type FilePair struct {
	Source      string
	Destination string
}

// ProcessResult represents the result of processing a single file
type ProcessResult struct {
	Success      bool
	FileName     string
	ChunkCount   int
	SpaceSavings SpaceSavings
	Error        error
}

// ProcessFile handles the core logic of processing a single file
func ProcessFile(ctx context.Context, pair FilePair, chunkSize int64, vaultRoot, passphrase string, progressMgr *progress.Manager, tags []string) ProcessResult {
	result := ProcessResult{
		FileName: filepath.Base(pair.Source),
	}

	// Determine path type and handle accordingly
	fileInfo, pathType, err := fs.GetPathInfo(pair.Source)
	if err != nil {
		result.Error = fmt.Errorf("failed to get path info: %v", err)
		return result
	}

	// Handle different path types
	var actualSourcePath string
	switch pathType {
	case fs.PathTypeFile:
		// Regular file - use as is
		actualSourcePath = pair.Source

	case fs.PathTypeSymlink:
		// Resolve symlink and verify target is a regular file
		targetPath, targetInfo, targetType, err := fs.ResolveSymlink(pair.Source)
		if err != nil {
			result.Error = fmt.Errorf("failed to resolve symlink: %v", err)
			return result
		}

		if targetType != fs.PathTypeFile {
			result.Error = fmt.Errorf("symlink target is not a regular file")
			return result
		}

		// Use the resolved target path for processing
		actualSourcePath = targetPath
		fileInfo = targetInfo

	default:
		result.Error = fmt.Errorf("unsupported file type")
		return result
	}

	// Get file size
	sizeInBytes := fileInfo.Size()

	// Process the file and store chunks
	chunkRefs, err := chunk.ChunkFile(ctx, actualSourcePath, chunkSize, vaultRoot, passphrase, progressMgr)
	if err != nil {
		result.Error = fmt.Errorf("chunking failed: %v", err)
		return result
	}

	result.ChunkCount = len(chunkRefs)

	// Create and store the file manifest
	destDir := filepath.Dir(pair.Destination)
	destFileName := filepath.Base(pair.Destination)

	// If the destination is just a filename (no directory), set destDir to empty
	if destDir == "." {
		destDir = ""
	} else if destDir != "" && !strings.HasPrefix(destDir, "/") {
		destDir = destDir + "/"
	}

	fileManifest := &config.FileManifest{
		FilePath:    destFileName,
		Size:        sizeInBytes,
		ModTime:     fileInfo.ModTime().Format(time.RFC3339),
		Chunks:      chunkRefs,
		Destination: destDir,
		AddedAt:     time.Now().UTC(),
		Tags:        tags,
	}

	// Save the manifest
	err = manifest.StoreFileManifest(vaultRoot, filepath.Base(pair.Source), fileManifest)
	if err != nil {
		if err.Error() == "skipped" {
			result.Error = fmt.Errorf("file was skipped")
			return result
		}
		result.Error = fmt.Errorf("manifest storage failed: %v", err)
		return result
	}

	// Calculate space savings for this file
	result.SpaceSavings = CalculateSpaceSavings(chunkRefs)
	result.Success = true

	return result
}
