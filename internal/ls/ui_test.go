package ls

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
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
func createTestManifest(path, dest string, size int64, tags []string, chunks []config.ChunkRef) config.FileManifest {
	return config.FileManifest{
		FilePath:    path,
		Destination: dest,
		Size:        size,
		ModTime:     time.Now().UTC().Format(time.RFC3339),
		Tags:        tags,
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

// Helper: build simple chunk index from files
func buildTestChunkIndex(files []config.FileManifest) map[string][]string {
	chunkRefs := make(map[string][]string)
	for _, f := range files {
		fp := f.Destination + f.FilePath
		for _, c := range f.Chunks {
			chunkID := c.Hash
			if chunkID == "" {
				chunkID = c.EncryptedHash
			}
			if chunkID == "" {
				continue
			}
			chunkRefs[chunkID] = append(chunkRefs[chunkID], fp)
		}
	}
	return chunkRefs
}

// ============================================================================
// FormatSharedWith TESTS
// ============================================================================

func TestFormatSharedWith_EmptyList(t *testing.T) {
	out := FormatSharedWith([]string{}, 10)
	if out != "" {
		t.Fatalf("expected empty string for empty list, got '%s'", out)
	}
}

func TestFormatSharedWith_SingleItem(t *testing.T) {
	list := []string{"file1.txt"}
	out := FormatSharedWith(list, 10)
	if out != "file1.txt" {
		t.Fatalf("expected 'file1.txt', got '%s'", out)
	}
}

func TestFormatSharedWith_BelowLimit(t *testing.T) {
	list := []string{"file1.txt", "file2.txt", "file3.txt"}
	out := FormatSharedWith(list, 10)
	expected := "file1.txt, file2.txt, file3.txt"
	if out != expected {
		t.Fatalf("expected '%s', got '%s'", expected, out)
	}
}

func TestFormatSharedWith_AtLimit(t *testing.T) {
	list := []string{"f1", "f2", "f3"}
	out := FormatSharedWith(list, 3)
	expected := "f1, f2, f3"
	if out != expected {
		t.Fatalf("expected '%s', got '%s'", expected, out)
	}
}

func TestFormatSharedWith_AboveLimit(t *testing.T) {
	list := []string{"f1", "f2", "f3", "f4", "f5"}
	out := FormatSharedWith(list, 3)
	if !strings.Contains(out, "f1, f2, f3") {
		t.Fatalf("expected first 3 items in output, got '%s'", out)
	}
	if !strings.Contains(out, "(+2 more)") {
		t.Fatalf("expected '(+2 more)' in output, got '%s'", out)
	}
}

func TestFormatSharedWith_LargeList(t *testing.T) {
	list := make([]string, 100)
	for i := 0; i < 100; i++ {
		list[i] = fmt.Sprintf("file%d.txt", i)
	}
	out := FormatSharedWith(list, 5)

	// Should show first 5 items
	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("file%d.txt", i)
		if !strings.Contains(out, expected) {
			t.Fatalf("expected '%s' in output, got '%s'", expected, out)
		}
	}

	// Should show (+95 more)
	if !strings.Contains(out, "(+95 more)") {
		t.Fatalf("expected '(+95 more)' in output, got '%s'", out)
	}
}

func TestFormatSharedWith_ZeroLimit(t *testing.T) {
	list := []string{"f1", "f2", "f3"}
	out := FormatSharedWith(list, 0)
	// With limit 0, should show (+3 more) with no visible items
	if !strings.Contains(out, "(+3 more)") {
		t.Fatalf("expected '(+3 more)' for zero limit, got '%s'", out)
	}
}

func TestFormatSharedWith_OneOverLimit(t *testing.T) {
	list := []string{"file1.txt", "file2.txt", "file3.txt"}
	out := FormatSharedWith(list, 2)
	if !strings.Contains(out, "file1.txt, file2.txt") {
		t.Fatalf("expected first 2 files, got '%s'", out)
	}
	if !strings.Contains(out, "(+1 more)") {
		t.Fatalf("expected '(+1 more)', got '%s'", out)
	}
}

// ============================================================================
// DisplayShortFormat TESTS
// ============================================================================

func TestDisplayShortFormat_EmptyList(t *testing.T) {
	var empty []config.FileManifest
	out := captureStdout(t, func() {
		DisplayShortFormat(empty, false, false, nil)
	})
	if out != "" {
		t.Fatalf("expected empty output for empty list, got '%s'", out)
	}
}

func TestDisplayShortFormat_SingleFile(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	expectedPath := "test/file.txt"
	if !strings.Contains(out, expectedPath) {
		t.Fatalf("expected '%s' in output, got '%s'", expectedPath, out)
	}
}

func TestDisplayShortFormat_MultipleFiles(t *testing.T) {
	f1 := createTestManifest("a.txt", "dir1/", 100, nil, nil)
	f2 := createTestManifest("b.txt", "dir2/", 200, nil, nil)
	f3 := createTestManifest("c.txt", "dir3/", 300, nil, nil)
	files := []config.FileManifest{f1, f2, f3}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

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

func TestDisplayShortFormat_WithTags(t *testing.T) {
	tags := []string{"important", "review", "urgent"}
	f1 := createTestManifest("file.txt", "test/", 100, tags, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, true, false, nil)
	})

	// Should show file path and tags in brackets
	if !strings.Contains(out, "test/file.txt") {
		t.Fatalf("expected file path in output, got '%s'", out)
	}
	if !strings.Contains(out, "[important, review, urgent]") {
		t.Fatalf("expected tags in brackets, got '%s'", out)
	}
}

func TestDisplayShortFormat_WithoutTags(t *testing.T) {
	tags := []string{"tag1", "tag2"}
	f1 := createTestManifest("file.txt", "test/", 100, tags, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	// Should not show tags when showTags=false
	if strings.Contains(out, "tag1") || strings.Contains(out, "tag2") {
		t.Fatalf("unexpected tags in output when showTags=false, got '%s'", out)
	}
	if strings.Contains(out, "[") {
		t.Fatalf("unexpected brackets in output, got '%s'", out)
	}
}

func TestDisplayShortFormat_EmptyTags(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, []string{}, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, true, false, nil)
	})

	// Should show path without tags
	if !strings.Contains(out, "test/file.txt") {
		t.Fatalf("expected file path in output, got '%s'", out)
	}
	// Should not show empty brackets
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines[0]) != len("test/file.txt") {
		// There should be nothing extra on the line
		if strings.Contains(lines[0], "[]") {
			t.Fatalf("unexpected empty brackets, got '%s'", out)
		}
	}
}

func TestDisplayShortFormat_NilTags(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, true, false, nil)
	})

	// Should handle nil tags gracefully
	if !strings.Contains(out, "test/file.txt") {
		t.Fatalf("expected file path in output, got '%s'", out)
	}
}

func TestDisplayShortFormat_DedupNoSharing(t *testing.T) {
	chunk := createTestChunk("unique1", 100)
	f1 := createTestManifest("file.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1}
	chunkRefs := buildTestChunkIndex(files)

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, true, chunkRefs)
	})

	if !strings.Contains(out, "shared_chunks: 0") {
		t.Fatalf("expected 'shared_chunks: 0' for non-shared file, got '%s'", out)
	}
	// Should not show shared_with when there's no sharing
	if strings.Contains(out, "shared_with:") {
		t.Fatalf("unexpected 'shared_with:' for non-shared file, got '%s'", out)
	}
}

func TestDisplayShortFormat_DedupWithSharing(t *testing.T) {
	chunk := createTestChunk("shared1", 100)
	f1 := createTestManifest("a.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	f2 := createTestManifest("b.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildTestChunkIndex(files)

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, true, chunkRefs)
	})

	if !strings.Contains(out, "shared_chunks:") {
		t.Fatalf("expected 'shared_chunks:' in output, got '%s'", out)
	}
	if !strings.Contains(out, "saved:") {
		t.Fatalf("expected 'saved:' in output, got '%s'", out)
	}
	if !strings.Contains(out, "shared_with:") {
		t.Fatalf("expected 'shared_with:' for shared chunks, got '%s'", out)
	}
}

func TestDisplayShortFormat_DedupNilChunkRefs(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil, nil)
	files := []config.FileManifest{f1}

	// Should not panic with nil chunkRefs
	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, true, nil)
	})

	if !strings.Contains(out, "test/file.txt") {
		t.Fatalf("expected file path even with nil chunkRefs, got '%s'", out)
	}
}

func TestDisplayShortFormat_DedupDisabled(t *testing.T) {
	chunk := createTestChunk("shared1", 100)
	f1 := createTestManifest("a.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	f2 := createTestManifest("b.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildTestChunkIndex(files)

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, chunkRefs)
	})

	// Should not show dedup stats when showDedup=false
	if strings.Contains(out, "shared_chunks:") {
		t.Fatalf("unexpected dedup stats when showDedup=false, got '%s'", out)
	}
}

func TestDisplayShortFormat_CombinedTagsAndDedup(t *testing.T) {
	chunk := createTestChunk("shared1", 100)
	tags := []string{"important"}
	f1 := createTestManifest("a.txt", "test/", 100, tags, []config.ChunkRef{chunk})
	f2 := createTestManifest("b.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildTestChunkIndex(files)

	out := captureStdout(t, func() {
		DisplayShortFormat(files, true, true, chunkRefs)
	})

	// Should show both tags and dedup stats
	if !strings.Contains(out, "[important]") {
		t.Fatalf("expected tags in output, got '%s'", out)
	}
	if !strings.Contains(out, "shared_chunks:") {
		t.Fatalf("expected dedup stats in output, got '%s'", out)
	}
}

func TestDisplayShortFormat_OutputFormat(t *testing.T) {
	f1 := createTestManifest("file.txt", "test/", 100, nil, nil)
	files := []config.FileManifest{f1}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line of output, got %d", len(lines))
	}

	// Verify clean output with just the path
	if lines[0] != "test/file.txt" {
		t.Fatalf("expected 'test/file.txt', got '%s'", lines[0])
	}
}

func TestDisplayShortFormat_MultipleFilesLineCount(t *testing.T) {
	f1 := createTestManifest("a.txt", "test/", 100, nil, nil)
	f2 := createTestManifest("b.txt", "test/", 200, nil, nil)
	f3 := createTestManifest("c.txt", "test/", 300, nil, nil)
	files := []config.FileManifest{f1, f2, f3}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines for 3 files, got %d", len(lines))
	}
}

func TestDisplayShortFormat_DedupIndentation(t *testing.T) {
	chunk := createTestChunk("shared1", 100)
	f1 := createTestManifest("a.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	f2 := createTestManifest("b.txt", "test/", 100, nil, []config.ChunkRef{chunk})
	files := []config.FileManifest{f1, f2}
	chunkRefs := buildTestChunkIndex(files)

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, true, chunkRefs)
	})

	// Dedup stats should be indented with space
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "shared_chunks:") {
			if !strings.HasPrefix(line, " ") {
				t.Fatalf("expected dedup line to start with space, got '%s'", line)
			}
		}
	}
}

func TestDisplayShortFormat_LargeFileList(t *testing.T) {
	files := make([]config.FileManifest, 100)
	for i := 0; i < 100; i++ {
		files[i] = createTestManifest(
			fmt.Sprintf("file%d.txt", i),
			"test/",
			int64(i*100),
			nil,
			nil,
		)
	}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 100 {
		t.Fatalf("expected 100 lines for 100 files, got %d", len(lines))
	}
}

func TestDisplayShortFormat_SpecialCharactersInPath(t *testing.T) {
	f1 := createTestManifest("file with spaces.txt", "test/", 100, nil, nil)
	f2 := createTestManifest("file-with-dashes.txt", "test/", 100, nil, nil)
	f3 := createTestManifest("file_with_underscores.txt", "test/", 100, nil, nil)
	files := []config.FileManifest{f1, f2, f3}

	out := captureStdout(t, func() {
		DisplayShortFormat(files, false, false, nil)
	})

	if !strings.Contains(out, "file with spaces.txt") {
		t.Fatalf("expected file with spaces in output")
	}
	if !strings.Contains(out, "file-with-dashes.txt") {
		t.Fatalf("expected file with dashes in output")
	}
	if !strings.Contains(out, "file_with_underscores.txt") {
		t.Fatalf("expected file with underscores in output")
	}
}
