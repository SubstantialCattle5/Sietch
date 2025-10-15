/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package add

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/util"
)

// VaultContext holds all the context needed for vault operations
type VaultContext struct {
	VaultRoot   string
	VaultConfig *config.VaultConfig
	ChunkSize   int64
	Passphrase  string
}

// SetupVaultContext sets up the vault context for add operations
func SetupVaultContext(cmd *cobra.Command) (*VaultContext, error) {
	// Find vault root
	vaultRoot, err := fs.FindVaultRoot()
	if err != nil {
		return nil, fmt.Errorf("not inside a vault: %v", err)
	}

	// Check if vault is initialized
	if !fs.IsVaultInitialized(vaultRoot) {
		return nil, fmt.Errorf("vault not initialized, run 'sietch init' first")
	}

	// Load vault configuration
	vaultConfig, err := config.LoadVaultConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %v", err)
	}

	// Parse chunk size
	chunkSize, err := util.ParseChunkSize(vaultConfig.Chunking.ChunkSize)
	if err != nil {
		// Fallback to default if parsing fails
		fmt.Printf("Warning: Invalid chunk size in configuration (%s). Using default (4MB).\n",
			vaultConfig.Chunking.ChunkSize)
		chunkSize = int64(constants.DefaultChunkSize) // Default to 4MB
	}

	// Get passphrase if needed for encryption
	passphrase, err := ui.GetPassphraseForVault(cmd, vaultConfig)
	if err != nil {
		return nil, err
	}

	return &VaultContext{
		VaultRoot:   vaultRoot,
		VaultConfig: vaultConfig,
		ChunkSize:   chunkSize,
		Passphrase:  passphrase,
	}, nil
}

// GetTagsFromCommand extracts tags from command flags
func GetTagsFromCommand(cmd *cobra.Command) ([]string, error) {
	tagsFlag, err := cmd.Flags().GetString("tags")
	if err != nil {
		return nil, fmt.Errorf("error parsing tags flag: %v", err)
	}

	tags := []string{}
	if tagsFlag != "" {
		// Simple split for now - could be enhanced to handle quoted tags with spaces
		tags = splitTags(tagsFlag)
	}

	return tags, nil
}

// splitTags splits a comma-separated string of tags, handling basic whitespace
func splitTags(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	// Split by comma and trim each tag
	tags := []string{}
	for _, tag := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}

	return tags
}
