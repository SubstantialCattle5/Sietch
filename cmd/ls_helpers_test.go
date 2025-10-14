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

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

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

// Helper: create test file manifest
func createTestManifest(path, dest string, size int64, chunks []config.ChunkRef) config.FileManifest {
	return config.FileManifest{
		FilePath:    path,
		Destination: dest,
		Size:        size,
		ModTime:     time.Now().UTC().Format(time.RFC3339),
		Chunks:      chunks,
	}
}

// Helper: create test chunk reference
func createTestChunk(hash string, encSize int64) config.ChunkRef {
	return config.ChunkRef{
		Hash:          hash,
		EncryptedSize: encSize,
	}
}

// ============================================================================
// EXISTING TESTS (PRESERVED)
// ============================================================================

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

	sharedChunks, savedBytes, sharedWith := dedup.ComputeDedupStatsForFile(f1, idx)
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
	sc, sb, sw := dedup.ComputeDedupStatsForFile(f3, idx)
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
	out := lsui.FormatSharedWith(list, 10)
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
		lsui.DisplayShortFormat(files, true, true, chunkRefs)
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
	_, _, sw := dedup.ComputeDedupStatsForFile(f1, idx)
	// sw should be sorted (we call sort.Strings), check monotonic property
	if !sort.StringsAreSorted(sw) {
		t.Fatalf("sharedWith not sorted: %v", sw)
	}
}

// ============================================================================
// NEW COMPREHENSIVE TESTS FOR PHASE 1
// ============================================================================

// ----------------------------------------------------------------------------
// filterAndSortFiles - Edge Cases
// ----------------------------------------------------------------------------

func TestFilterAndSortFiles_EmptyInput(t *testing.T) {
	var empty []config.FileManifest
	out := filterAndSortFiles(empty, "", "path")
	if len(out) != 0 {
		t.Fatalf("expected 0 files from empty input, got %d", len(out))
	}
}

func TestFilterAndSortFiles_NoMatches(t *testing.T) {
	f1 := createTestManifest("a.txt", "docs/", 100, nil)
	f2 := createTestManifest("b.txt", "data/", 200, nil)
	files := []config.FileManifest{f1, f2}

	out := filterAndSortFiles(files, "images/", "path")
	if len(out) != 0 {
		t.Fatalf("expected 0 files matching 'images/', got %d", len(out))
	}
}

func TestFilterAndSortFiles_AllSortTypes(t *testing.T) {
	now := time.Now().UTC()
	// Create files with different timestamps
	f1 := config.FileManifest{
		FilePath:    "zebra.txt",
		Destination: "test/",
		Size:        50,
		ModTime:     now.Add(-2 * time.Hour).Format(time.RFC3339),
	}
	f2 := config.FileManifest{
		FilePath:    "alpha.txt",
		Destination: "test/",
		Size:        200,
		ModTime:     now.Add(-1 * time.Hour).Format(time.RFC3339),
	}
	f3 := config.FileManifest{
		FilePath:    "beta.txt",
		Destination: "test/",
		Size:        100,
		ModTime:     now.Format(time.RFC3339),
	}
	files := []config.FileManifest{f1, f2, f3}

	// Test sort by name
	byName := filterAndSortFiles(files, "", "name")
	if byName[0].FilePath != "alpha.txt" || byName[1].FilePath != "beta.txt" || byName[2].FilePath != "zebra.txt" {
		t.Fatalf("sort by name failed: %v", []string{byName[0].FilePath, byName[1].FilePath, byName[2].FilePath})
	}

	// Test sort by size (descending)
	bySize := filterAndSortFiles(files, "", "size")
	if bySize[0].Size != 200 || bySize[1].Size != 100 || bySize[2].Size != 50 {
		t.Fatalf("sort by size failed: %v", []int64{bySize[0].Size, bySize[1].Size, bySize[2].Size})
	}

	// Test sort by time (most recent first)
	byTime := filterAndSortFiles(files, "", "time")
	if byTime[0].FilePath != "beta.txt" || byTime[1].FilePath != "alpha.txt" || byTime[2].FilePath != "zebra.txt" {
		t.Fatalf("sort by time failed: %v", []string{byTime[0].FilePath, byTime[1].FilePath, byTime[2].FilePath})
	}

	// Test default/path sort
	byPath := filterAndSortFiles(files, "", "unknown")
	// All have same destination, so order by destination comparison
	if len(byPath) != 3 {
		t.Fatalf("default sort failed, expected 3 files got %d", len(byPath))
	}
}

func TestFilterAndSortFiles_PartialPathMatch(t *testing.T) {
	f1 := createTestManifest("file1.txt", "docs/subdir/", 100, nil)
	f2 := createTestManifest("file2.txt", "docs/", 200, nil)
	f3 := createTestManifest("file3.txt", "data/", 300, nil)
	files := []config.FileManifest{f1, f2, f3}

	// Filter by "docs/" should match both docs/ and docs/subdir/
	out := filterAndSortFiles(files, "docs/", "path")
	if len(out) != 2 {
		t.Fatalf("expected 2 files with prefix 'docs/', got %d", len(out))
	}
}

func TestFilterAndSortFiles_CaseInsensitiveSort(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil)
	files := []config.FileManifest{f1}

	// Test various case combinations for sort parameter
	testCases := []string{"NAME", "Size", "TIME", "Path"}
	for _, sortBy := range testCases {
		out := filterAndSortFiles(files, "", sortBy)
		if len(out) != 1 {
			t.Fatalf("sort by '%s' failed, expected 1 file got %d", sortBy, len(out))
		}
	}
}

// ----------------------------------------------------------------------------
// buildChunkIndex - Edge Cases
// ----------------------------------------------------------------------------

func TestBuildChunkIndex_EmptyInput(t *testing.T) {
	var empty []config.FileManifest
	idx := buildChunkIndex(empty)
	if len(idx) != 0 {
		t.Fatalf("expected empty index from empty input, got %d entries", len(idx))
	}
}

func TestBuildChunkIndex_NoChunks(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, []config.ChunkRef{})
	files := []config.FileManifest{f1}

	idx := buildChunkIndex(files)
	if len(idx) != 0 {
		t.Fatalf("expected empty index when files have no chunks, got %d entries", len(idx))
	}
}

func TestBuildChunkIndex_EncryptedHashFallback(t *testing.T) {
	// Chunk with empty Hash but has EncryptedHash
	chunk := config.ChunkRef{
		Hash:          "",
		EncryptedHash: "encrypted123",
		EncryptedSize: 100,
	}
	f1 := createTestManifest("file.txt", "test/", 100, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1}

	idx := buildChunkIndex(files)
	if len(idx["encrypted123"]) != 1 {
		t.Fatalf("expected fallback to EncryptedHash, got %d refs for 'encrypted123'", len(idx["encrypted123"]))
	}
}

func TestBuildChunkIndex_SkipEmptyChunks(t *testing.T) {
	// Chunks with both Hash and EncryptedHash empty
	chunk1 := config.ChunkRef{Hash: "", EncryptedHash: "", EncryptedSize: 100}
	chunk2 := config.ChunkRef{Hash: "valid123", EncryptedHash: "", EncryptedSize: 200}
	f1 := createTestManifest("file.txt", "test/", 300, []config.ChunkRef{chunk1, chunk2})
	files := []config.FileManifest{f1}

	idx := buildChunkIndex(files)
	if len(idx) != 1 {
		t.Fatalf("expected 1 valid chunk in index, got %d", len(idx))
	}
	if len(idx["valid123"]) != 1 {
		t.Fatalf("expected 1 ref for valid chunk, got %d", len(idx["valid123"]))
	}
}

func TestBuildChunkIndex_MultipleFilesSharedChunks(t *testing.T) {
	chunk1 := createTestChunk("shared1", 100)
	chunk2 := createTestChunk("unique1", 200)
	chunk3 := createTestChunk("shared1", 100) // same as chunk1

	f1 := createTestManifest("a.txt", "dir1/", 100, []config.ChunkRef{chunk1, chunk2})
	f2 := createTestManifest("b.txt", "dir2/", 100, []config.ChunkRef{chunk3})
	files := []config.FileManifest{f1, f2}

	idx := buildChunkIndex(files)

	if len(idx["shared1"]) != 2 {
		t.Fatalf("expected 2 refs for shared chunk, got %d", len(idx["shared1"]))
	}
	if len(idx["unique1"]) != 1 {
		t.Fatalf("expected 1 ref for unique chunk, got %d", len(idx["unique1"]))
	}
}

func TestBuildChunkIndex_FullPathConstruction(t *testing.T) {
	chunk := createTestChunk("hash1", 100)
	f1 := createTestManifest("file.txt", "nested/path/", 100, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1}

	idx := buildChunkIndex(files)
	refs := idx["hash1"]
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	expectedPath := "nested/path/file.txt"
	if refs[0] != expectedPath {
		t.Fatalf("expected path '%s', got '%s'", expectedPath, refs[0])
	}
}

// ----------------------------------------------------------------------------
// displayLongFormat - Edge Cases
// ----------------------------------------------------------------------------

func TestDisplayLongFormat_EmptyFileList(t *testing.T) {
	var empty []config.FileManifest
	out := captureStdout(t, func() {
		displayLongFormat(empty, false, false, nil)
	})
	// Should only contain header
	if !strings.Contains(out, "SIZE") || !strings.Contains(out, "MODIFIED") {
		t.Fatalf("expected header in empty output, got: %s", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected only header line for empty input, got %d lines", len(lines))
	}
}

func TestDisplayLongFormat_WithTags(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil)
	f1.Tags = []string{"important", "review"}
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		displayLongFormat(files, true, false, nil)
	})

	if !strings.Contains(out, "TAGS") {
		t.Fatalf("expected TAGS column header, got: %s", out)
	}
	if !strings.Contains(out, "important") || !strings.Contains(out, "review") {
		t.Fatalf("expected tags in output, got: %s", out)
	}
}

func TestDisplayLongFormat_WithoutTags(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		displayLongFormat(files, false, false, nil)
	})

	if strings.Contains(out, "TAGS") {
		t.Fatalf("unexpected TAGS column when showTags=false, got: %s", out)
	}
}

func TestDisplayLongFormat_HeaderVerification(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		displayLongFormat(files, false, false, nil)
	})

	requiredHeaders := []string{"SIZE", "MODIFIED", "CHUNKS", "PATH"}
	for _, header := range requiredHeaders {
		if !strings.Contains(out, header) {
			t.Fatalf("expected header '%s' in output, got: %s", header, out)
		}
	}
}

func TestDisplayLongFormat_TimeFormatting(t *testing.T) {
	now := time.Now().UTC()
	f1 := config.FileManifest{
		FilePath:    "file.txt",
		Destination: "test/",
		Size:        100,
		ModTime:     now.Format(time.RFC3339),
	}
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		displayLongFormat(files, false, false, nil)
	})

	// Check for expected time format: "2006-01-02 15:04:05"
	expectedFormat := now.Format("2006-01-02")
	if !strings.Contains(out, expectedFormat) {
		t.Fatalf("expected time format containing '%s' in output, got: %s", expectedFormat, out)
	}
}

func TestDisplayLongFormat_ChunkCount(t *testing.T) {
	chunks := []config.ChunkRef{
		createTestChunk("c1", 100),
		createTestChunk("c2", 200),
		createTestChunk("c3", 300),
	}
	f1 := createTestManifest("file.txt", "test/", 600, chunks)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		displayLongFormat(files, false, false, nil)
	})

	// Should show chunk count of 3
	if !strings.Contains(out, "3") {
		t.Fatalf("expected chunk count '3' in output, got: %s", out)
	}
}

func TestDisplayLongFormat_DedupWithNoSharing(t *testing.T) {
	chunk := createTestChunk("unique1", 100)
	f1 := createTestManifest("file.txt", "test/", 100, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1}
	chunkRefs := buildChunkIndex(files)

	out := captureStdout(t, func() {
		displayLongFormat(files, false, true, chunkRefs)
	})

	if !strings.Contains(out, "shared_chunks: 0") {
		t.Fatalf("expected 'shared_chunks: 0' for non-shared file, got: %s", out)
	}
	if strings.Contains(out, "shared_with:") {
		t.Fatalf("unexpected 'shared_with:' for non-shared file, got: %s", out)
	}
}

func TestDisplayLongFormat_DedupWithSharing(t *testing.T) {
	chunk := createTestChunk("shared1", 100)
	f1 := createTestManifest("a.txt", "test/", 100, []config.ChunkRef{chunk})
	f2 := createTestManifest("b.txt", "test/", 100, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildChunkIndex(files)

	out := captureStdout(t, func() {
		displayLongFormat(files, false, true, chunkRefs)
	})

	if !strings.Contains(out, "shared_chunks:") {
		t.Fatalf("expected dedup stats in output, got: %s", out)
	}
	if !strings.Contains(out, "shared_with:") {
		t.Fatalf("expected 'shared_with:' for shared chunks, got: %s", out)
	}
}

func TestDisplayLongFormat_NilChunkRefsWithDedupFlag(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil)
	files := []config.FileManifest{f1}

	// Should not panic with nil chunkRefs
	out := captureStdout(t, func() {
		displayLongFormat(files, false, true, nil)
	})

	// Should still show header but no dedup stats
	if !strings.Contains(out, "SIZE") {
		t.Fatalf("expected header even with nil chunkRefs, got: %s", out)
	}
}

func TestDisplayLongFormat_MultipleFiles(t *testing.T) {
	f1 := createTestManifest("a.txt", "dir1/", 100, nil)
	f2 := createTestManifest("b.txt", "dir2/", 200, nil)
	f3 := createTestManifest("c.txt", "dir3/", 300, nil)
	files := []config.FileManifest{f1, f2, f3}

	out := captureStdout(t, func() {
		displayLongFormat(files, false, false, nil)
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Should have header + 3 file lines
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines (header + 3 files), got %d", len(lines))
	}

	// Verify all files appear
	if !strings.Contains(out, "dir1/a.txt") {
		t.Fatalf("expected 'dir1/a.txt' in output")
	}
	if !strings.Contains(out, "dir2/b.txt") {
		t.Fatalf("expected 'dir2/b.txt' in output")
	}
	if !strings.Contains(out, "dir3/c.txt") {
		t.Fatalf("expected 'dir3/c.txt' in output")
	}
}