package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestManifestSummaryMode(t *testing.T) {
	vaultRoot := createManifestTestVault(t)
	writeManifestFixture(t, vaultRoot, "docs.report.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks:
  - hash: chunk-1
    size: 42
    index: 0
tags: ["docs"]
added_at: "2025-01-02T03:04:05Z"
`)

	output, err := runManifestForTest(t, vaultRoot, map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(output, "Vault Name: Test Vault") {
		t.Fatalf("expected vault name in output, got: %s", output)
	}
	if !strings.Contains(output, "Manifest Count: 1") {
		t.Fatalf("expected manifest count in output, got: %s", output)
	}
}

func TestManifestFileModeWithJSON(t *testing.T) {
	vaultRoot := createManifestTestVault(t)
	writeManifestFixture(t, vaultRoot, "docs.report.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks:
  - hash: chunk-1
    size: 42
    index: 0
tags: ["docs", "important"]
added_at: "2025-01-02T03:04:05Z"
last_synced: "2025-01-03T03:04:05Z"
last_verified: "2025-01-04T03:04:05Z"
`)

	output, err := runManifestForTest(t, vaultRoot, map[string]string{"file": "report.txt", "json": "true"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("expected valid json, got error: %v and output: %s", unmarshalErr, output)
	}

	stableFields := []string{"manifest_id", "file", "destination", "full_path", "chunk_count", "chunk_refs", "modified_at", "added_at"}
	for _, field := range stableFields {
		if _, ok := payload[field]; !ok {
			t.Fatalf("expected field %q in json payload: %v", field, payload)
		}
	}

	if payload["full_path"] != "docs/report.txt" {
		t.Fatalf("expected full_path docs/report.txt, got %#v", payload["full_path"])
	}
}

func TestManifestSummaryJSONStableFields(t *testing.T) {
	vaultRoot := createManifestTestVault(t)
	writeManifestFixture(t, vaultRoot, "docs.report.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks:
  - hash: chunk-1
    size: 42
    index: 0
added_at: "2025-01-02T03:04:05Z"
`)

	output, err := runManifestForTest(t, vaultRoot, map[string]string{"json": "true"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("expected valid json, got error: %v and output: %s", unmarshalErr, output)
	}

	stableFields := []string{"vault_name", "vault_path", "encryption_type", "chunk_size", "manifest_count"}
	for _, field := range stableFields {
		if _, ok := payload[field]; !ok {
			t.Fatalf("expected field %q in json payload: %v", field, payload)
		}
	}
}

func TestManifestVerifySuccessAndFailurePaths(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vaultRoot := createManifestTestVault(t)
		writeManifestFixture(t, vaultRoot, "docs.report.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks:
  - hash: chunk-1
    size: 42
    index: 0
added_at: "2025-01-02T03:04:05Z"
`)

		_, err := runManifestForTest(t, vaultRoot, map[string]string{"verify": "true"})
		if err != nil {
			t.Fatalf("expected verify success, got error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		vaultRoot := createManifestTestVault(t)
		writeManifestFixture(t, vaultRoot, "docs.report.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks: []
added_at: "2025-01-02T03:04:05Z"
`)
		writeManifestFixture(t, vaultRoot, "docs.report-copy.txt", `file: report.txt
size: 42
mtime: "2025-01-02T03:04:05Z"
destination: docs/
chunks:
  - hash: chunk-2
    size: 42
    index: 0
added_at: "2025-01-02T03:04:05Z"
`)

		output, err := runManifestForTest(t, vaultRoot, map[string]string{"verify": "true", "json": "true"})
		if err == nil {
			t.Fatalf("expected verify failure to return error")
		}

		var payload map[string]any
		if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
			t.Fatalf("expected valid json, got error: %v and output: %s", unmarshalErr, output)
		}
		if payload["passed"] != false {
			t.Fatalf("expected passed=false, got: %#v", payload["passed"])
		}
	})
}

func createManifestTestVault(t *testing.T) string {
	t.Helper()

	vaultRoot := t.TempDir()
	manifestsDir := filepath.Join(vaultRoot, ".sietch", "manifests")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		t.Fatalf("failed to create manifests directory: %v", err)
	}

	vaultYAML := `name: Test Vault
vault_id: test-vault-id
created_at: 2025-01-01T00:00:00Z
schema_version: 1
encryption:
  type: aes
  key_path: .sietch/keys/master.key
  passphrase_protected: false
chunking:
  strategy: fixed
  chunk_size: 4MB
  hash_algorithm: sha256
compression: gzip
deduplication:
  enabled: true
  strategy: content
  min_chunk_size: 1KB
  max_chunk_size: 64MB
  gc_threshold: 1000
  index_enabled: true
sync:
  mode: local
  enabled: false
metadata:
  author: tester
  tags: []
`

	if err := os.WriteFile(filepath.Join(vaultRoot, "vault.yaml"), []byte(vaultYAML), 0o644); err != nil {
		t.Fatalf("failed to write vault.yaml: %v", err)
	}

	return vaultRoot
}

func writeManifestFixture(t *testing.T, vaultRoot, id, content string) {
	t.Helper()
	path := filepath.Join(vaultRoot, ".sietch", "manifests", id+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write manifest fixture %s: %v", id, err)
	}
}

func runManifestForTest(t *testing.T, vaultRoot string, flags map[string]string) (string, error) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if chdirErr := os.Chdir(vaultRoot); chdirErr != nil {
		t.Fatalf("failed to chdir to vault root: %v", chdirErr)
	}
	defer os.Chdir(originalDir)

	cmd := &cobra.Command{}
	buffer := &bytes.Buffer{}
	cmd.SetOut(buffer)
	cmd.Flags().String("file", "", "")
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("verify", false, "")

	for name, value := range flags {
		if setErr := cmd.Flags().Set(name, value); setErr != nil {
			t.Fatalf("failed to set flag %s=%s: %v", name, value, setErr)
		}
	}

	runErr := runManifestCommand(cmd, nil)
	return strings.TrimSpace(buffer.String()), runErr
}
