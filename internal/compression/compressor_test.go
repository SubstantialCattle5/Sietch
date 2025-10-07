package compression

import (
	"bytes"
	"testing"

	"github.com/substantialcattle5/sietch/internal/constants"
)

func TestCompressionAlgorithms(t *testing.T) {
	data := []byte("Divyansh Raj Soni testing LZ4 compression support")

	algorithms := []string{
		constants.CompressionTypeNone,
		constants.CompressionTypeGzip,
		constants.CompressionTypeZstd,
		constants.CompressionTypeLZ4, // ✅ new test
	}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			compressed, err := CompressData(data, algo)
			if err != nil {
				t.Fatalf("Compression failed for %s: %v", algo, err)
			}

			decompressed, err := DecompressData(compressed, algo)
			if err != nil {
				t.Fatalf("Decompression failed for %s: %v", algo, err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Fatalf("%s: decompressed data mismatch", algo)
			}
		})
	}
}
