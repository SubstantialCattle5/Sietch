package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
)

// Helper: capture stdout while running fn()
func captureStdout(t *testing.T, fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()

	// Close writer and restore stdout before reading
	w.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		os.Stdout = old
		t.Fatalf("copy: %v", err)
	}
	os.Stdout = old
	return buf.String()
}

func TestFilterAndSortFiles_Basic(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	f1 := config.FileManifest{FilePath: "a.txt", Destination: "docs/", Size: 100, ModTime: now}
	f2 := config.FileManifest{FilePath: "b.txt", Destination: "docs/", Size: 200, ModTime: now}
	f3 := config.FileManifest{FilePath: "c.txt", Destination: "data/", Size: 50, ModTime: now}

	files := []config.FileManifest{f1, f2, f3}

	// sort by name
	out := filterAndSortFiles(files, "", "name")
	if out[0].FilePath != "a.txt" || out[1].FilePath != "b.txt" || out[2].FilePath != "c.txt" {
		t.Fatalf("unexpected order by name: %v", []string{out[0].FilePath, out[1].FilePath, out[2].FilePath})
	}

	// sort by size (desc)
	out = filterAndSortFiles(files, "", "size")
	if out[0].Size < out[1].Size || out[1].Size < out[2].Size {
		t.Fatalf("unexpected order by size: %v", []int64{out[0].Size, out[1].Size, out[2].Size})
	}

	// filter by destination prefix
	out = filterAndSortFiles(files, "docs/", "path")
	if len(out) != 2 {
		t.Fatalf("expected 2 files in docs/, got %d", len(out))
	}
}

func TestBuildChunkIndexAndComputeDedupStats(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)

	// file1 has chunks c1 and c2
	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "test/",
		Size:        1024,
		ModTime:     now,
		Chunks: []config.ChunkRef{
			{Hash: "c1", EncryptedSize: 128},
			{Hash: "c2", EncryptedSize: 256},
		},
	}
	// file2 shares c1
	f2 := config.FileManifest{
		FilePath:    "b.txt",
		Destination: "test/",
		Size:        1024,
		ModTime:     now,
		Chunks: []config.ChunkRef{
			{Hash: "c1", EncryptedSize: 128},
		},
	}
	// file3 no share
	f3 := config.FileManifest{
		FilePath:    "c.txt",
		Destination: "other/",
		Size:        512,
		ModTime:     now,
		Chunks: []config.ChunkRef{
			{Hash: "c3", EncryptedSize: 64},
		},
	}

	files := []config.FileManifest{f1, f2, f3}

	idx := buildChunkIndex(files)

	// verify chunk index
	if len(idx["c1"]) != 2 {
		t.Fatalf("expected c1 refs length 2, got %d", len(idx["c1"]))
	}
	if len(idx["c2"]) != 1 {
		t.Fatalf("expected c2 refs length 1, got %d", len(idx["c2"]))
	}

	sharedChunks, savedBytes, sharedWith := computeDedupStatsForFile(f1, idx)
	if sharedChunks != 1 {
		t.Fatalf("expected sharedChunks 1 for f1, got %d", sharedChunks)
	}
	if savedBytes != 128 {
		t.Fatalf("expected savedBytes 128 for f1, got %d", savedBytes)
	}
	if len(sharedWith) != 1 {
		t.Fatalf("expected sharedWith length 1, got %d", len(sharedWith))
	}
	if sharedWith[0] != "test/b.txt" {
		t.Fatalf("expected shared with test/b.txt got %v", sharedWith)
	}

	// file with no shared chunks
	sc, sb, sw := computeDedupStatsForFile(f3, idx)
	if sc != 0 || sb != 0 || len(sw) != 0 {
		t.Fatalf("expected no shared chunks for f3, got sc=%d sb=%d sw=%v", sc, sb, sw)
	}
}

func TestFormatSharedWith_Truncation(t *testing.T) {
	list := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		// use numeric suffixes to avoid rune/int confusion
		list = append(list, fmt.Sprintf("file%d", i))
	}
	out := formatSharedWith(list, 10)
	if !strings.Contains(out, "(+2 more)") {
		t.Fatalf("expected truncation info (+2 more) in '%s'", out)
	}
}

func TestDisplayShortAndLongFormat_OutputContainsStats(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)

	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "test/",
		Size:        100,
		ModTime:     now,
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 128}},
	}
	f2 := config.FileManifest{
		FilePath:    "b.txt",
		Destination: "test/",
		Size:        200,
		ModTime:     now,
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 128}},
	}
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildChunkIndex(files)

	// short format capture
	outShort := captureStdout(t, func() {
		displayShortFormat(files, true, true, chunkRefs)
	})
	if !strings.Contains(outShort, "shared_chunks:") || !strings.Contains(outShort, "saved:") {
		t.Fatalf("short output missing dedup info: %s", outShort)
	}

	// long format capture
	outLong := captureStdout(t, func() {
		displayLongFormat(files, false, true, chunkRefs)
	})
	if !strings.Contains(outLong, "SIZE") || !strings.Contains(outLong, "shared_chunks:") {
		t.Fatalf("long output missing dedup info: %s", outLong)
	}
}

func TestBuildChunkIndex_DeterministicOrder(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)

	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "x/",
		Size:        10,
		ModTime:     now,
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 10}},
	}
	f2 := config.FileManifest{
		FilePath:    "b.txt",
		Destination: "y/",
		Size:        20,
		ModTime:     now,
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 10}},
	}
	files := []config.FileManifest{f1, f2}
	idx := buildChunkIndex(files)

	// ensure entries are present
	if len(idx["c1"]) != 2 {
		t.Fatalf("expected 2 refs for c1; got %d", len(idx["c1"]))
	}

	// ensure computeDedupStatsForFile sorts sharedWith deterministically
	_, _, sw := computeDedupStatsForFile(f1, idx)
	// sw should be sorted (we call sort.Strings), check monotonic property
	if !sort.StringsAreSorted(sw) {
		t.Fatalf("sharedWith not sorted: %v", sw)
	}
}
