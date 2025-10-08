package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	dedup "github.com/substantialcattle5/sietch/internal/deduplication"
	lsui "github.com/substantialcattle5/sietch/internal/ls"
)

// Test tags that contain commas and special characters are printed as-is in short output.
func TestTagsWithCommasAndSpecialChars(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	f := config.FileManifest{
		FilePath:    "tagged.txt",
		Destination: "docs/",
		Size:        42,
		ModTime:     now,
		Tags:        []string{"one,two", "three/four", "weird,tag"},
	}
	out := captureStdout(t, func() {
		lsui.DisplayShortFormat([]config.FileManifest{f}, true, false, nil)
	})
	// All tags should appear exactly (commas preserved inside tags)
	for _, tg := range f.Tags {
		if !strings.Contains(out, tg) {
			t.Fatalf("expected tag %q present in output: %s", tg, out)
		}
	}
}

// Test unicode filenames are handled and printed correctly.
func TestUnicodeFilenames(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	unicodeName := "ファイル_测试_файл.txt"
	f := config.FileManifest{
		FilePath:    unicodeName,
		Destination: "u/",
		Size:        10,
		ModTime:     now,
	}
	out := captureStdout(t, func() {
		lsui.DisplayShortFormat([]config.FileManifest{f}, false, false, nil)
	})
	if !strings.Contains(out, unicodeName) {
		t.Fatalf("unicode filename not present in output: %s", out)
	}
}

// Unknown/invalid sort should fall back to default (path) without panicking.
func TestInvalidSortFlagFallsBack(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	a := config.FileManifest{FilePath: "a", Destination: "d/", Size: 1, ModTime: now}
	b := config.FileManifest{FilePath: "b", Destination: "c/", Size: 2, ModTime: now}
	files := []config.FileManifest{a, b}

	out := filterAndSortFiles(files, "", "IAmNotARealSort")
	if len(out) != 2 {
		t.Fatalf("expected 2 files after fallback sort, got %d", len(out))
	}
}

// Files with no chunks should produce zero dedup stats.
func TestEmptyChunksProduceZeroDedupStats(t *testing.T) {
	f := config.FileManifest{FilePath: "nochunks", Destination: "x/", Chunks: []config.ChunkRef{}}
	idx := buildChunkIndex([]config.FileManifest{f})
	sc, sb, sw := dedup.ComputeDedupStatsForFile(f, idx)
	if sc != 0 || sb != 0 || len(sw) != 0 {
		t.Fatalf("expected zero dedup stats for file with no chunks, got sc=%d sb=%d sw=%v", sc, sb, sw)
	}
}

// Large sharedWith list should be truncated by FormatSharedWith and include "(+N more)".
func TestLargeSharedWithTruncationBehavior(t *testing.T) {
	// create a list of 25 entries
	list := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		list = append(list, fmt.Sprintf("file-%02d", i))
	}
	out := lsui.FormatSharedWith(list, 10)
	if !strings.Contains(out, "(+15 more)") {
		t.Fatalf("expected truncation for large sharedWith list, got: %s", out)
	}
	// ensure first visible items are present
	if !strings.HasPrefix(out, "file-00, file-01") {
		t.Fatalf("expected first items visible, got: %s", out)
	}
}

// Test many chunks (lightweight stress) — ensure compute finishes and returns expected counts.
// This is not super heavy to keep unit test fast.
func TestManyChunksLightStress(t *testing.T) {
	// Build a file with 100 chunks, where every even chunk is shared with another file.
	const n = 100
	chunksA := make([]config.ChunkRef, 0, n)
	chunksB := make([]config.ChunkRef, 0, n/2)
	for i := 0; i < n; i++ {
		h := fmt.Sprintf("h-%04d", i)
		if i%2 == 0 {
			// shared chunk: both files reference same hash
			chunksA = append(chunksA, config.ChunkRef{Hash: h, EncryptedSize: 64})
			chunksB = append(chunksB, config.ChunkRef{Hash: h, EncryptedSize: 64})
		} else {
			// unique chunk
			chunksA = append(chunksA, config.ChunkRef{Hash: h, EncryptedSize: 32})
		}
	}
	fA := config.FileManifest{FilePath: "A", Destination: "t/", Chunks: chunksA}
	fB := config.FileManifest{FilePath: "B", Destination: "t/", Chunks: chunksB}
	idx := buildChunkIndex([]config.FileManifest{fA, fB})

	sharedChunks, savedBytes, sharedWith := dedup.ComputeDedupStatsForFile(fA, idx)
	// Expect n/2 shared (even indices)
	if sharedChunks != n/2 {
		t.Fatalf("expected %d shared chunks, got %d", n/2, sharedChunks)
	}
	// savedBytes should be > 0
	if savedBytes <= 0 {
		t.Fatalf("expected positive saved bytes, got %d", savedBytes)
	}
	// sharedWith should contain B
	found := false
	for _, s := range sharedWith {
		if strings.Contains(s, "B") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sharedWith to include B, got %v", sharedWith)
	}
}

// If chunkRefs maps a chunk only to the file itself, it should not be considered shared.
func TestChunkRefsWithSelfOnlyNotShared(t *testing.T) {
	f := config.FileManifest{
		FilePath:    "self.txt",
		Destination: "d/",
		Chunks:      []config.ChunkRef{{Hash: "solo"}},
	}
	// build artificial index mapping the chunk to the same file path only
	idx := map[string][]string{
		"solo": {"d/self.txt"},
	}
	sc, sb, sw := dedup.ComputeDedupStatsForFile(f, idx)
	if sc != 0 || sb != 0 || len(sw) != 0 {
		t.Fatalf("expected no shared chunks when index only references same file, got sc=%d sb=%d sw=%v", sc, sb, sw)
	}
}

// Sorting by time when some ModTime values are invalid should not panic and must include all files.
func TestSortByTimeWithInvalidModTimes(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	valid := config.FileManifest{FilePath: "v.txt", Destination: "p/", Size: 1, ModTime: now}
	invalid := config.FileManifest{FilePath: "i.txt", Destination: "p/", Size: 2, ModTime: "not-a-time"}
	files := []config.FileManifest{valid, invalid}
	out := filterAndSortFiles(files, "", "time")
	if len(out) != 2 {
		t.Fatalf("expected 2 files after time sort (even with invalid ModTime), got %d", len(out))
	}
}
