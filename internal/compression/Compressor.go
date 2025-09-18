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
		if _, err := io.Copy(&buf, reader); err != nil {
			return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
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
		return decompressed, nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}
