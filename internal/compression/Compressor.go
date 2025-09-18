package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// CompressData compresses data according to the specified compression algorithm
func CompressData(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case constants.CompressionTypeNone:
		return data, nil
	case constants.CompressionTypeGzip:
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write gzip data: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		return buf.Bytes(), nil
	case constants.CompressionTypeZstd:
		encoder, err := zstd.NewWriter(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
		}
		defer encoder.Close()
		return encoder.EncodeAll(data, nil), nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// DecompressData decompresses data according to the specified compression algorithm
func DecompressData(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case constants.CompressionTypeNone:
		return data, nil
	case constants.CompressionTypeGzip:
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		var buf bytes.Buffer
		// Use io.CopyN to limit decompression size and prevent decompression bombs
		n, err := io.CopyN(&buf, reader, constants.MaxDecompressionSize)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
		}

		// Check if we hit the limit (potential decompression bomb)
		if n == constants.MaxDecompressionSize {
			// Try to read one more byte to see if there's more data
			var extraByte [1]byte
			if _, err := reader.Read(extraByte[:]); err == nil {
				return nil, fmt.Errorf("decompressed data exceeds maximum size limit (%d bytes) - potential decompression bomb", constants.MaxDecompressionSize)
			}
		}
		return buf.Bytes(), nil
	case constants.CompressionTypeZstd:
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(data, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress zstd data: %w", err)
		}

		// Check decompressed size to prevent decompression bombs
		if len(decompressed) > constants.MaxDecompressionSize {
			return nil, fmt.Errorf("decompressed data exceeds maximum size limit (%d bytes) - potential decompression bomb", constants.MaxDecompressionSize)
		}

		return decompressed, nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}
