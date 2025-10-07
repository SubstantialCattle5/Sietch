package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4" // ✅ Added LZ4 import
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

	case constants.CompressionTypeLZ4: // ✅ New Case Added
		var buf bytes.Buffer
		writer := lz4.NewWriter(&buf)
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write lz4 data: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close lz4 writer: %w", err)
		}
		return buf.Bytes(), nil

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
		n, err := io.CopyN(&buf, reader, constants.MaxDecompressionSize)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
		}

		if n == constants.MaxDecompressionSize {
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

		if len(decompressed) > constants.MaxDecompressionSize {
			return nil, fmt.Errorf("decompressed data exceeds maximum size limit (%d bytes) - potential decompression bomb", constants.MaxDecompressionSize)
		}

		return decompressed, nil

	case constants.CompressionTypeLZ4: // ✅ New Case Added
		reader := lz4.NewReader(bytes.NewReader(data))
		var buf bytes.Buffer
		n, err := io.CopyN(&buf, reader, constants.MaxDecompressionSize)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to decompress lz4 data: %w", err)
		}

		if n == constants.MaxDecompressionSize {
			var extraByte [1]byte
			if _, err := reader.Read(extraByte[:]); err == nil {
				return nil, fmt.Errorf("decompressed data exceeds maximum size limit (%d bytes) - potential decompression bomb", constants.MaxDecompressionSize)
			}
		}
		return buf.Bytes(), nil

	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}
