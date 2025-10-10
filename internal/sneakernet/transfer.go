package sneakernet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	"gopkg.in/yaml.v3"
)

// Analyze performs analysis of what would be transferred
func (st *SneakTransfer) Analyze() (*SneakAnalysis, error) {
	// Initialize managers
	sourceManager, err := config.NewManager(st.SourceVault)
	if err != nil {
		return nil, fmt.Errorf("failed to create source manager: %v", err)
	}

	destManager, err := config.NewManager(st.DestVault)
	if err != nil {
		return nil, fmt.Errorf("failed to create dest manager: %v", err)
	}

	// Load manifests
	sourceManifest, err := sourceManager.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load source manifest: %v", err)
	}

	destManifest, err := destManager.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load dest manifest: %v", err)
	}

	// Analyze differences
	analysis := &SneakAnalysis{
		NewFiles:        []config.FileManifest{},
		NewChunks:       []string{},
		DuplicateChunks: []string{},
		Conflicts:       []FileConflict{},
	}

	// Build maps for efficient lookup
	destFileMap := make(map[string]config.FileManifest)
	destChunkMap := make(map[string]bool)

	for _, file := range destManifest.Files {
		destFileMap[file.FilePath] = file
		for _, chunk := range file.Chunks {
			destChunkMap[chunk.Hash] = true
			if chunk.EncryptedHash != "" {
				destChunkMap[chunk.EncryptedHash] = true
			}
		}
	}

	// Process source files
	for _, sourceFile := range sourceManifest.Files {
		// Apply file pattern filters
		if !st.matchesFilePatterns(sourceFile.FilePath) {
			continue
		}

		if destFile, exists := destFileMap[sourceFile.FilePath]; exists {
			// File exists in destination - check for conflicts
			if st.filesConflict(sourceFile, destFile) {
				conflict := FileConflict{
					FilePath:   sourceFile.FilePath,
					SourceInfo: sourceFile,
					DestInfo:   destFile,
					Resolution: ConflictResolution{},
				}

				// Auto-resolve if requested
				if st.AutoResolve {
					conflict.Resolution.Action = "rename"
					conflict.Resolution.NewName = st.generateRenameName(sourceFile.FilePath)
				}

				analysis.Conflicts = append(analysis.Conflicts, conflict)
			}
			// If no conflict, file already exists and is identical - skip
		} else {
			// New file
			analysis.NewFiles = append(analysis.NewFiles, sourceFile)
		}

		// Process chunks for this file
		for _, chunk := range sourceFile.Chunks {
			chunkExists := destChunkMap[chunk.Hash]
			if chunk.EncryptedHash != "" && destChunkMap[chunk.EncryptedHash] {
				chunkExists = true
			}

			if chunkExists {
				analysis.DuplicateChunks = append(analysis.DuplicateChunks, chunk.Hash)
				analysis.DuplicateSize += chunk.Size
			} else {
				analysis.NewChunks = append(analysis.NewChunks, chunk.Hash)
				analysis.TransferSize += chunk.Size
			}
		}

		analysis.TotalSize += sourceFile.Size
	}

	return analysis, nil
}

// Execute performs the actual sneakernet transfer
func (st *SneakTransfer) Execute() (*TransferResult, error) {
	// Re-run analysis to get current state
	analysis, err := st.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze transfer: %v", err)
	}

	startTime := time.Now()
	result := &TransferResult{
		Conflicts: analysis.Conflicts,
		Errors:    []string{},
	}

	// Create managers for the actual transfer
	sourceManager, err := config.NewManager(st.SourceVault)
	if err != nil {
		return nil, fmt.Errorf("failed to create source manager: %v", err)
	}

	destManager, err := config.NewManager(st.DestVault)
	if err != nil {
		return nil, fmt.Errorf("failed to create dest manager: %v", err)
	}

	if st.DryRun {
		return result, nil
	}

	// Transfer chunks
	if st.Verbose {
		fmt.Printf("Transferring %d chunks...\n", len(analysis.NewChunks))
	}

	for i, chunkHash := range analysis.NewChunks {
		if st.Verbose && i%10 == 0 {
			fmt.Printf("Progress: %d/%d chunks\n", i, len(analysis.NewChunks))
		}

		err := st.transferChunkWithManagers(chunkHash, sourceManager, destManager)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to transfer chunk %s: %v", chunkHash, err)
			result.Errors = append(result.Errors, errorMsg)
			if st.Verbose {
				fmt.Printf("Error: %s\n", errorMsg)
			}
			continue
		}

		result.ChunksTransferred++
	}

	result.ChunksSkipped = len(analysis.DuplicateChunks)

	// Transfer manifests for new files and resolved conflicts
	err = st.transferManifestsWithAnalysis(result, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer manifests: %v", err)
	}

	// Calculate bytes transferred
	result.BytesTransferred = analysis.TransferSize
	result.Duration = time.Since(startTime)

	// Rebuild references in destination vault
	err = destManager.RebuildReferences()
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild references: %v", err)
	}

	return result, nil
}

// transferChunkWithManagers copies a chunk from source to destination using provided managers
func (st *SneakTransfer) transferChunkWithManagers(chunkHash string, sourceManager, destManager *config.Manager) error {
	// For encrypted vaults, chunks are stored with encrypted_hash as filename
	// but we receive the plain hash. We need to try both plain and encrypted hash.

	// First, try to use the plain hash (works for unencrypted vaults and backwards compatibility)
	exists, err := destManager.ChunkExists(chunkHash)
	if err != nil {
		return fmt.Errorf("failed to check chunk existence: %v", err)
	}

	storageHash := chunkHash
	if exists {
		// Chunk exists with plain hash, use it
	} else {
		// Chunk doesn't exist with plain hash. For encrypted vaults, try to find it with encrypted hash.
		// We need to check if this is an encrypted vault and find the encrypted hash.
		if encryptedHash := st.findEncryptedHashForChunk(chunkHash, sourceManager); encryptedHash != "" {
			storageHash = encryptedHash

			// Check if chunk exists with encrypted hash
			exists, err = destManager.ChunkExists(encryptedHash)
			if err != nil {
				return fmt.Errorf("failed to check chunk existence with encrypted hash: %v", err)
			}
		}
	}

	if exists {
		return nil // Already exists, skip
	}

	// Read chunk from source using the appropriate hash
	sourceChunkData, err := sourceManager.GetChunk(storageHash)
	if err != nil {
		return fmt.Errorf("failed to read source chunk: %v", err)
	}

	// Store chunk in destination using the appropriate hash
	err = destManager.StoreChunk(storageHash, sourceChunkData)
	if err != nil {
		return fmt.Errorf("failed to store chunk: %v", err)
	}

	return nil
}

// findEncryptedHashForChunk finds the encrypted hash for a chunk given its plain hash
func (st *SneakTransfer) findEncryptedHashForChunk(plainHash string, sourceManager *config.Manager) string {
	// Get the source vault configuration to check if it's encrypted
	vaultConfig, err := sourceManager.GetConfig()
	if err != nil {
		if st.Verbose {
			fmt.Printf("Warning: Failed to get vault config to check encryption: %v\n", err)
		}
		return ""
	}

	// If vault is not encrypted, chunks are stored with plain hash
	if vaultConfig.Encryption.Type == "" || vaultConfig.Encryption.Type == "none" {
		return ""
	}

	// For encrypted vaults, we need to find the chunk metadata
	// Get the manifest to find chunk references
	manifest, err := sourceManager.GetManifest()
	if err != nil {
		if st.Verbose {
			fmt.Printf("Warning: Failed to get manifest to find encrypted hash: %v\n", err)
		}
		return ""
	}

	// Search through all files and chunks to find the matching plain hash
	for _, file := range manifest.Files {
		for _, chunk := range file.Chunks {
			if chunk.Hash == plainHash && chunk.EncryptedHash != "" {
				return chunk.EncryptedHash
			}
		}
	}

	return ""
}

// transferManifestsWithAnalysis handles manifest transfer for new files and conflicts
func (st *SneakTransfer) transferManifestsWithAnalysis(result *TransferResult, analysis *SneakAnalysis) error {
	manifestsDir := filepath.Join(st.DestVault, ".sietch", "manifests")

	// Transfer new files
	for _, newFile := range analysis.NewFiles {
		err := st.writeFileManifest(manifestsDir, newFile)
		if err != nil {
			return fmt.Errorf("failed to write manifest for %s: %v", newFile.FilePath, err)
		}
		result.FilesTransferred++
	}

	// Transfer resolved conflicts
	for _, conflict := range analysis.Conflicts {
		switch conflict.Resolution.Action {
		case "overwrite":
			err := st.writeFileManifest(manifestsDir, conflict.SourceInfo)
			if err != nil {
				return fmt.Errorf("failed to overwrite manifest for %s: %v", conflict.FilePath, err)
			}
			result.FilesTransferred++

		case "rename":
			// Create a new manifest with the renamed file path
			renamedFile := conflict.SourceInfo
			renamedFile.FilePath = conflict.Resolution.NewName
			renamedFile.Destination = conflict.Resolution.NewName

			err := st.writeFileManifest(manifestsDir, renamedFile)
			if err != nil {
				return fmt.Errorf("failed to write renamed manifest for %s: %v", conflict.Resolution.NewName, err)
			}
			result.FilesTransferred++

		case "skip":
			// Do nothing
		}
	}

	return nil
}

// writeFileManifest writes a file manifest to the manifests directory
func (st *SneakTransfer) writeFileManifest(manifestsDir string, fileManifest config.FileManifest) error {
	// Ensure manifests directory exists
	err := os.MkdirAll(manifestsDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create manifests directory: %v", err)
	}

	// Generate manifest filename (use a safe filename based on the file path)
	manifestName := st.generateManifestFilename(fileManifest.FilePath)
	manifestPath := filepath.Join(manifestsDir, manifestName)

	// Write manifest file (this would need to be implemented to match the existing format)
	// For now, we'll use a placeholder implementation
	return st.saveFileManifest(manifestPath, fileManifest)
}

// Helper functions

// matchesFilePatterns checks if a file path matches the specified patterns
func (st *SneakTransfer) matchesFilePatterns(filePath string) bool {
	// If no patterns specified, include all files
	if len(st.FilePatterns) == 0 && len(st.ExcludePatterns) == 0 {
		return true
	}

	// Check exclude patterns first
	for _, pattern := range st.ExcludePatterns {
		if st.matchesPattern(filePath, pattern) {
			return false
		}
	}

	// If include patterns specified, file must match at least one
	if len(st.FilePatterns) > 0 {
		for _, pattern := range st.FilePatterns {
			if st.matchesPattern(filePath, pattern) {
				return true
			}
		}
		return false
	}

	return true
}

// matchesPattern performs simple pattern matching (supports * wildcards)
func (st *SneakTransfer) matchesPattern(filePath, pattern string) bool {
	// Simple pattern matching implementation
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		// Handle wildcard patterns
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(filePath, parts[0]) && strings.HasSuffix(filePath, parts[1])
		}
	}

	// Exact match or substring match
	matched, err := filepath.Match(pattern, filePath)
	return strings.Contains(filePath, pattern) || (err == nil && matched)
}

// filesConflict determines if two file manifests represent conflicting files
func (st *SneakTransfer) filesConflict(source, dest config.FileManifest) bool {
	// Files conflict if they have the same path but different content
	return source.ContentHash != dest.ContentHash
}

// generateRenameName creates a new name for conflicting files
func (st *SneakTransfer) generateRenameName(originalPath string) string {
	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	timestamp := time.Now().Format("2006-01-02-150405")
	newName := fmt.Sprintf("%s_from_source_%s%s", name, timestamp, ext)

	if dir == "." {
		return newName
	}
	return filepath.Join(dir, newName)
}

// generateManifestFilename creates a safe filename for manifest files
func (st *SneakTransfer) generateManifestFilename(filePath string) string {
	// Replace path separators and unsafe characters
	safe := strings.ReplaceAll(filePath, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	return safe + ".yaml"
}

// saveFileManifest saves a file manifest
func (st *SneakTransfer) saveFileManifest(manifestPath string, fileManifest config.FileManifest) error {
	// Create the file
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %v", err)
	}
	defer file.Close()

	// Encode the manifest to YAML with proper indentation
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(fileManifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %v", err)
	}

	return nil
}
