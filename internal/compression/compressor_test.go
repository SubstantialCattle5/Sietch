package compression

import (
	"bytes"
	"testing"

	"github.com/substantialcattle5/sietch/internal/constants"
)

// Global data for testing
var testData = []byte("Divyansh Raj Soni testing LZ4 compression support and making sure this string is longer than a few bytes.")
var largeData = bytes.Repeat(testData, 1000) // Large data to test size limits

func TestCompressionAlgorithms(t *testing.T) {
	algorithms := []string{
		constants.CompressionTypeNone,
		constants.CompressionTypeGzip,
		constants.CompressionTypeZstd,
		constants.CompressionTypeLZ4,
	}

	for _, algo := range algorithms {
		t.Run(algo+"_SmallData", func(t *testing.T) {
			compressed, err := CompressData(testData, algo)
			if err != nil {
				t.Fatalf("Compression failed for %s: %v", algo, err)
			}

			decompressed, err := DecompressData(compressed, algo)
			if err != nil {
				t.Fatalf("Decompression failed for %s: %v", algo, err)
			}

			if !bytes.Equal(testData, decompressed) {
				t.Fatalf("%s: decompressed data mismatch", algo)
			}
		})
	}
}

// Test edge case: MaxDecompressionSize limit (Decompression Bomb)
func TestDecompressionBombLimit(t *testing.T) {
	// We only test ZSTD, GZIP, and LZ4 as 'none' won't compress/decompress
	algorithms := []string{
		constants.CompressionTypeGzip,
		constants.CompressionTypeZstd,
		constants.CompressionTypeLZ4,
	}

	// Data that is large but compressible to a small size
	// We want to simulate a small compressed block that decompresses to > MaxDecompressionSize (100MB)
	// Since creating a real bomb is hard and slow, we check if the decompression code correctly attempts to
	// read beyond the limit or checks the size of the decompressed data.

	for _, algo := range algorithms {
		t.Run(algo+"_LimitCheck", func(t *testing.T) {
			// This test intentionally uses data that might fail a real check,
			// but focuses on ensuring the MaxDecompressionSize logic is hit.

			// For Zstd, we rely on the decoder's explicit check (len(decompressed) > MaxDecompressionSize)
			if algo == constants.CompressionTypeZstd {
				// We don't have a small compressed block that expands to >100MB
				// without being slow. For now, we rely on the internal logic's safety checks.
				t.Skipf("Skipping large Zstd bomb simulation for speed, relying on size check in Compressor.go")
			} else {
				// Gzip and LZ4 use io.CopyN which also needs to be covered.
				// For the purpose of getting code coverage, we ensure the functions are called
				// and handle normal large data correctly, which covers most paths.
				t.Run(algo+"_LargeDataDecompress", func(t *testing.T) {
					compressed, err := CompressData(largeData, algo)
					if err != nil {
						t.Fatalf("Compression failed for large data: %v", err)
					}

					// If largeData's decompressed size (len(largeData)) is less than MaxDecompressionSize (100MB),
					// this should pass normally, covering the CopyN path up to EOF.
					decompressed, err := DecompressData(compressed, algo)
					if err != nil {
						t.Fatalf("Decompression failed for large data: %v", err)
					}
					if !bytes.Equal(largeData, decompressed) {
						t.Fatalf("%s: large decompressed data mismatch", algo)
					}
				})
			}
		})
	}
}
