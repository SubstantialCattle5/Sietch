package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	manifestbuilder "github.com/substantialcattle5/sietch/internal/manifest"
)

type manifestSummaryOutput struct {
	VaultName      string `json:"vault_name"`
	VaultPath      string `json:"vault_path"`
	EncryptionType string `json:"encryption_type"`
	KeyPath        string `json:"key_path,omitempty"`
	ChunkSize      string `json:"chunk_size"`
	ManifestCount  int    `json:"manifest_count"`
}

type fileManifestOutput struct {
	ManifestID   string            `json:"manifest_id"`
	File         string            `json:"file"`
	Destination  string            `json:"destination"`
	FullPath     string            `json:"full_path"`
	ChunkCount   int               `json:"chunk_count"`
	ChunkRefs    []config.ChunkRef `json:"chunk_refs"`
	Tags         []string          `json:"tags,omitempty"`
	ModifiedAt   string            `json:"modified_at"`
	AddedAt      string            `json:"added_at"`
	LastSynced   string            `json:"last_synced,omitempty"`
	LastVerified string            `json:"last_verified,omitempty"`
}

type verifyOutput struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures"`
}

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Inspect vault and file manifests",
	Long: `Inspect vault metadata and file manifest entries.

Examples:
  sietch manifest
  sietch manifest --file docs/report.pdf
  sietch manifest --file report.pdf --json
  sietch manifest --verify`,
	RunE: runManifestCommand,
}

func runManifestCommand(cmd *cobra.Command, _ []string) error {
	fileArg, _ := cmd.Flags().GetString("file")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	verify, _ := cmd.Flags().GetBool("verify")

	vaultRoot, err := fs.FindVaultRoot()
	if err != nil {
		return fmt.Errorf("not inside a vault: %v", err)
	}

	if verify {
		result := verifyVaultManifests(vaultRoot)
		if jsonOutput {
			if err := writeJSON(cmd, result); err != nil {
				return err
			}
			if !result.Passed {
				return errors.New("manifest verification failed")
			}
			return nil
		}
		for _, failure := range result.Failures {
			fmt.Fprintf(cmd.OutOrStdout(), "FAIL: %s\n", failure)
		}
		if result.Passed {
			fmt.Fprintln(cmd.OutOrStdout(), "Manifest verification passed")
			return nil
		}
		return errors.New("manifest verification failed")
	}

	if fileArg != "" {
		manifestID, mf, err := resolveFileManifest(vaultRoot, fileArg)
		if err != nil {
			return err
		}
		output := buildFileManifestOutput(manifestID, mf)
		if jsonOutput {
			return writeJSON(cmd, output)
		}
		printFileManifestSummary(cmd, output)
		return nil
	}

	summary, err := buildManifestSummary(vaultRoot)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(cmd, summary)
	}
	printManifestSummary(cmd, summary)
	return nil
}

func buildManifestSummary(vaultRoot string) (manifestSummaryOutput, error) {
	vaultCfg, err := manifestbuilder.LoadVaultConfig(vaultRoot)
	if err != nil {
		return manifestSummaryOutput{}, fmt.Errorf("failed to load vault metadata: %w", err)
	}

	manifestIDs, err := manifestbuilder.ListFileManifests(vaultRoot)
	if err != nil {
		return manifestSummaryOutput{}, fmt.Errorf("failed to list file manifests: %w", err)
	}

	return manifestSummaryOutput{
		VaultName:      vaultCfg.Name,
		VaultPath:      vaultRoot,
		EncryptionType: vaultCfg.Encryption.Type,
		KeyPath:        vaultCfg.Encryption.KeyPath,
		ChunkSize:      vaultCfg.Chunking.ChunkSize,
		ManifestCount:  len(manifestIDs),
	}, nil
}

func resolveFileManifest(vaultRoot, fileArg string) (string, *config.FileManifest, error) {
	manifestIDs, err := manifestbuilder.ListFileManifests(vaultRoot)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list file manifests: %w", err)
	}
	if len(manifestIDs) == 0 {
		return "", nil, fmt.Errorf("no files found in vault")
	}

	sort.Strings(manifestIDs)
	for _, id := range manifestIDs {
		if id == fileArg {
			mf, loadErr := manifestbuilder.LoadFileManifest(vaultRoot, id)
			if loadErr != nil {
				return "", nil, fmt.Errorf("failed to load manifest '%s': %w", id, loadErr)
			}
			return id, mf, nil
		}
	}

	targetBase := filepath.Base(fileArg)
	var suggestions []string
	for _, id := range manifestIDs {
		mf, loadErr := manifestbuilder.LoadFileManifest(vaultRoot, id)
		if loadErr != nil {
			continue
		}
		fullPath := mf.Destination + mf.FilePath
		if fullPath == fileArg || mf.FilePath == fileArg || filepath.Base(mf.FilePath) == fileArg || filepath.Base(fullPath) == fileArg {
			return id, mf, nil
		}
		if filepath.Base(fullPath) == targetBase {
			suggestions = append(suggestions, fullPath)
		}
	}

	if len(suggestions) > 0 {
		return "", nil, fmt.Errorf("no file found matching '%s'. Did you mean one of: %v", fileArg, suggestions)
	}

	return "", nil, fmt.Errorf("no file found matching '%s'. Use 'sietch ls' to see available files", fileArg)
}

func buildFileManifestOutput(manifestID string, mf *config.FileManifest) fileManifestOutput {
	out := fileManifestOutput{
		ManifestID:  manifestID,
		File:        mf.FilePath,
		Destination: mf.Destination,
		FullPath:    mf.Destination + mf.FilePath,
		ChunkCount:  len(mf.Chunks),
		ChunkRefs:   mf.Chunks,
		Tags:        mf.Tags,
		ModifiedAt:  mf.ModTime,
		AddedAt:     mf.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if !mf.LastSynced.IsZero() {
		out.LastSynced = mf.LastSynced.Format("2006-01-02T15:04:05Z07:00")
	}
	if !mf.LastVerified.IsZero() {
		out.LastVerified = mf.LastVerified.Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

func verifyVaultManifests(vaultRoot string) verifyOutput {
	failures := make([]string, 0)

	manifestIDs, err := manifestbuilder.ListFileManifests(vaultRoot)
	if err != nil {
		failures = append(failures, fmt.Sprintf("unable to list manifests: %v", err))
		return verifyOutput{Passed: false, Failures: failures}
	}

	seenIDs := make(map[string]bool)
	seenFullPaths := make(map[string]string)
	for _, id := range manifestIDs {
		if seenIDs[id] {
			failures = append(failures, fmt.Sprintf("duplicate manifest ID: %s", id))
			continue
		}
		seenIDs[id] = true

		mf, loadErr := manifestbuilder.LoadFileManifest(vaultRoot, id)
		if loadErr != nil {
			failures = append(failures, fmt.Sprintf("missing or unreadable manifest '%s': %v", id, loadErr))
			continue
		}

		fullPath := mf.Destination + mf.FilePath
		if priorID, exists := seenFullPaths[fullPath]; exists {
			failures = append(failures, fmt.Sprintf("duplicate manifest ID for file '%s': %s and %s", fullPath, priorID, id))
		} else {
			seenFullPaths[fullPath] = id
		}

		if len(mf.Chunks) == 0 {
			failures = append(failures, fmt.Sprintf("zero-chunk manifest entry: %s", id))
		}
	}

	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")
	for _, id := range manifestIDs {
		manifestPath := filepath.Join(manifestsDir, id+".yaml")
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			failures = append(failures, fmt.Sprintf("missing manifest file for ID '%s'", id))
		}
	}

	return verifyOutput{Passed: len(failures) == 0, Failures: failures}
}

func printManifestSummary(cmd *cobra.Command, summary manifestSummaryOutput) {
	fmt.Fprintf(cmd.OutOrStdout(), "Vault Name: %s\n", summary.VaultName)
	fmt.Fprintf(cmd.OutOrStdout(), "Vault Path: %s\n", summary.VaultPath)
	fmt.Fprintf(cmd.OutOrStdout(), "Encryption: %s\n", summary.EncryptionType)
	if summary.KeyPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Key Path: %s\n", summary.KeyPath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Chunk Size: %s\n", summary.ChunkSize)
	fmt.Fprintf(cmd.OutOrStdout(), "Manifest Count: %d\n", summary.ManifestCount)
}

func printFileManifestSummary(cmd *cobra.Command, output fileManifestOutput) {
	fmt.Fprintf(cmd.OutOrStdout(), "Manifest ID: %s\n", output.ManifestID)
	fmt.Fprintf(cmd.OutOrStdout(), "Destination: %s\n", output.Destination)
	fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", output.File)
	fmt.Fprintf(cmd.OutOrStdout(), "Full Path: %s\n", output.FullPath)
	fmt.Fprintf(cmd.OutOrStdout(), "Chunks: %d\n", output.ChunkCount)
	if len(output.Tags) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Tags: %s\n", strings.Join(output.Tags, ", "))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Modified: %s\n", output.ModifiedAt)
	fmt.Fprintf(cmd.OutOrStdout(), "Added: %s\n", output.AddedAt)
	if output.LastSynced != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Last Synced: %s\n", output.LastSynced)
	}
	if output.LastVerified != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Last Verified: %s\n", output.LastVerified)
	}
}

func writeJSON(cmd *cobra.Command, payload any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func init() {
	rootCmd.AddCommand(manifestCmd)
	manifestCmd.Flags().String("file", "", "File path, file name, or manifest ID to inspect")
	manifestCmd.Flags().Bool("json", false, "Output machine-readable JSON")
	manifestCmd.Flags().Bool("verify", false, "Run manifest consistency checks")
}
