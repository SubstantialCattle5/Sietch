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
	dedup "github.com/substantialcattle5/sietch/internal/deduplication"
	lsui "github.com/substantialcattle5/sietch/internal/ls"
)

// captureStdout captures stdout produced by fn() and returns it.
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

func TestFilterAndSortFiles_BasicAndEdgeCases(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	// three files in two destinations
	f1 := config.FileManifest{FilePath: "a.txt", Destination: "docs/", Size: 100, ModTime: now}
	f2 := config.FileManifest{FilePath: "b.txt", Destination: "docs/", Size: 200, ModTime: now}
	f3 := config.FileManifest{FilePath: "c.txt", Destination: "data/", Size: 50, ModTime: now}

	files := []config.FileManifest{f1, f2, f3}

	// sort by name
	out := filterAndSortFiles(files, "", "name")
	if len(out) != 3 || out[0].FilePath != "a.txt" || out[1].FilePath != "b.txt" || out[2].FilePath != "c.txt" {
		t.Fatalf("unexpected order by name: %v", []string{out[0].FilePath, out[1].FilePath, out[2].FilePath})
	}

	// sort by size (desc)
	out = filterAndSortFiles(files, "", "size")
	if !(out[0].Size >= out[1].Size && out[1].Size >= out[2].Size) {
		t.Fatalf("unexpected order by size: %v", []int64{out[0].Size, out[1].Size, out[2].Size})
	}

	// filter by destination prefix
	out = filterAndSortFiles(files, "docs/", "path")
	if len(out) != 2 {
		t.Fatalf("expected 2 files in docs/, got %d", len(out))
	}

	// empty input
	empty := filterAndSortFiles(nil, "", "name")
	if len(empty) != 0 {
		t.Fatalf("expected empty result for nil input, got %d", len(empty))
	}

	// identical names & sizes - stable-ish behaviour: ensure it doesn't panic and returns same count
	dup1 := config.FileManifest{FilePath: "same.txt", Destination: "x/", Size: 10, ModTime: now}
	dup2 := config.FileManifest{FilePath: "same.txt", Destination: "y/", Size: 10, ModTime: now}
	out = filterAndSortFiles([]config.FileManifest{dup1, dup2}, "", "name")
	if len(out) != 2 {
		t.Fatalf("expected 2 dup files, got %d", len(out))
	}
}

func TestBuildChunkIndex_FallbackAndSkipping(t *testing.T) {
	time.Now().UTC().Format(time.RFC3339)

	// chunk with normal Hash
	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "t/",
		Chunks:      []config.ChunkRef{{Hash: "h1"}},
	}
	// chunk with empty Hash but EncryptedHash present
	f2 := config.FileManifest{
		FilePath:    "b.txt",
		Destination: "t/",
		Chunks:      []config.ChunkRef{{Hash: "", EncryptedHash: "eh1"}},
	}
	// chunk with both empty -> should be skipped
	f3 := config.FileManifest{
		FilePath:    "c.txt",
		Destination: "t/",
		Chunks:      []config.ChunkRef{{Hash: "", EncryptedHash: ""}},
	}

	idx := buildChunkIndex([]config.FileManifest{f1, f2, f3})

	if len(idx["h1"]) != 1 {
		t.Fatalf("expected h1 to map to 1 file, got %d", len(idx["h1"]))
	}
	if len(idx["eh1"]) != 1 {
		t.Fatalf("expected eh1 to map to 1 file, got %d", len(idx["eh1"]))
	}
	if _, ok := idx[""]; ok {
		t.Fatalf("expected empty chunk id not present in index")
	}
}

func TestComputeDedupStatsForFile_SizesAndSharedWith(t *testing.T) {
	time.Now().UTC().Format(time.RFC3339)

	// a.txt has two chunks: c1 (shared) and c2 (unique)
	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "test/",
		Chunks: []config.ChunkRef{
			{Hash: "c1", EncryptedSize: 128},
			{Hash: "c2", Size: 256},
		},
	}
	// b.txt shares c1
	f2 := config.FileManifest{
		FilePath:    "b.txt",
		Destination: "test/",
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 128}},
	}
	// c.txt no share
	f3 := config.FileManifest{
		FilePath:    "c.txt",
		Destination: "other/",
		Chunks:      []config.ChunkRef{{Hash: "c3", EncryptedSize: 64}},
	}

	files := []config.FileManifest{f1, f2, f3}
	idx := buildChunkIndex(files)

	sharedChunks, savedBytes, sharedWith := dedup.ComputeDedupStatsForFile(f1, idx)
	if sharedChunks != 1 {
		t.Fatalf("expected 1 shared chunk, got %d", sharedChunks)
	}
	if savedBytes != 128 {
		t.Fatalf("expected saved bytes 128 (from EncryptedSize), got %d", savedBytes)
	}
	if len(sharedWith) != 1 || sharedWith[0] != "test/b.txt" {
		t.Fatalf("unexpected sharedWith: %v", sharedWith)
	}

	// file with fallback to Size (EncryptedSize zero)
	f4 := config.FileManifest{
		FilePath:    "d.txt",
		Destination: "test/",
		Chunks:      []config.ChunkRef{{Hash: "c1", Size: 512}},
	}
	files2 := []config.FileManifest{f4, f2}
	idx2 := buildChunkIndex(files2)
	sc, sb, sw := dedup.ComputeDedupStatsForFile(f4, idx2)
	if sc != 1 {
		t.Fatalf("expected 1 shared chunk (f4), got %d", sc)
	}
	if sb != 512 {
		t.Fatalf("expected saved bytes 512 (from Size), got %d", sb)
	}
	if len(sw) != 1 {
		t.Fatalf("expected sharedWith length 1, got %d", len(sw))
	}

	// fallback when both sizes are 0 -> default chunk size used
	f5 := config.FileManifest{
		FilePath:    "e.txt",
		Destination: "x/",
		Chunks:      []config.ChunkRef{{Hash: "x1"}},
	}
	f6 := config.FileManifest{
		FilePath:    "g.txt",
		Destination: "x/",
		Chunks:      []config.ChunkRef{{Hash: "x1"}},
	}
	idx3 := buildChunkIndex([]config.FileManifest{f5, f6})
	sc2, sb2, _ := dedup.ComputeDedupStatsForFile(f5, idx3)
	const defaultChunk = 4 * 1024 * 1024
	if sc2 != 1 || sb2 != defaultChunk {
		t.Fatalf("expected default chunk size saved (%d), got sc=%d sb=%d", defaultChunk, sc2, sb2)
	}
}

func TestFormatSharedWith_TruncationAndEmpty(t *testing.T) {
	// empty
	if got := lsui.FormatSharedWith([]string{}, 3); got != "" {
		t.Fatalf("expected empty string for empty list, got %q", got)
	}

	// under limit
	list := []string{"a", "b"}
	if got := lsui.FormatSharedWith(list, 5); got != "a, b" {
		t.Fatalf("unexpected format under limit: %q", got)
	}

	// over limit
	long := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		long = append(long, fmt.Sprintf("file%d", i))
	}
	got := lsui.FormatSharedWith(long, 10)
	if !strings.Contains(got, "(+2 more)") {
		t.Fatalf("expected truncation marker in %q", got)
	}
}

func TestDisplayShortAndLongFormat_OutputAndEdgeCases(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)

	// normal files for display
	f1 := config.FileManifest{
		FilePath:    "a.txt",
		Destination: "test/",
		Size:        100,
		ModTime:     now,
		Chunks:      []config.ChunkRef{{Hash: "c1", EncryptedSize: 128}},
		Tags:        []string{"alpha", "beta"},
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

	// short format shows path and tags and dedup info
	outShort := captureStdout(t, func() {
		lsui.DisplayShortFormat(files, true, true, chunkRefs)
	})
	if !strings.Contains(outShort, "test/a.txt") || !strings.Contains(outShort, "alpha") {
		t.Fatalf("short output missing filename or tags: %s", outShort)
	}
	if !strings.Contains(outShort, "shared_chunks:") || !strings.Contains(outShort, "saved:") {
		t.Fatalf("short output missing dedup info: %s", outShort)
	}

	// long format: use displayLongFormat from cmd package
	outLong := captureStdout(t, func() {
		displayLongFormat(files, true, true, chunkRefs)
	})
	if !strings.Contains(outLong, "SIZE") || !strings.Contains(outLong, "shared_chunks:") {
		t.Fatalf("long output missing expected headers or dedup info: %s", outLong)
	}

	// edge: invalid ModTime should not panic and should print zero-time
	bad := config.FileManifest{
		FilePath:    "badtime.txt",
		Destination: "x/",
		Size:        10,
		ModTime:     "not-a-time",
	}
	outBad := captureStdout(t, func() {
		displayLongFormat([]config.FileManifest{bad}, false, false, nil)
	})
	if !strings.Contains(outBad, "0001-01-01") {
		t.Fatalf("expected zero-time in output for invalid ModTime, got: %s", outBad)
	}

	// long filename handling (should appear in output)
	longName := strings.Repeat("longname_", 10) + ".txt"
	lf := config.FileManifest{FilePath: longName, Destination: "p/", Size: 1, ModTime: now}
	outLongName := captureStdout(t, func() {
		displayLongFormat([]config.FileManifest{lf}, false, false, nil)
	})
	if !strings.Contains(outLongName, longName) {
		t.Fatalf("long filename not present in output")
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
	_, _, sw := dedup.ComputeDedupStatsForFile(f1, idx)
	if !sort.StringsAreSorted(sw) {
		t.Fatalf("sharedWith not sorted deterministically: %v", sw)
	}
}
