package chunk

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/util"
	"github.com/zeebo/blake3"
)

// formatChunkInfo formats and prints chunk processing information
func FormatChunkInfo(chunkCount int, bytesRead int, chunkHash string, vaultConfig config.VaultConfig, chunkDataToProcess []byte, deduplicated bool, encrypted bool) {
	compressionInfo := ""
	if vaultConfig.Compression != "none" {
		compressionInfo = fmt.Sprintf(" (compressed with %s: %s -> %s)",
			vaultConfig.Compression,
			util.HumanReadableSize(int64(bytesRead)),
			util.HumanReadableSize(int64(len(chunkDataToProcess))))
	}

	dedupInfo := ""
	if deduplicated {
		dedupInfo = " [deduplicated]"
	}

	displayHash := chunkHash
	if encrypted && len(chunkHash) > HashDisplayLength {
		displayHash = chunkHash[:HashDisplayLength]
	}

	encryptionInfo := ""
	if encrypted {
		encryptionInfo = " (encrypted)"
	}

	fmt.Printf("Chunk %d: %s bytes, hash: %s%s%s%s\n",
		chunkCount,
		util.HumanReadableSize(int64(bytesRead)),
		displayHash,
		encryptionInfo,
		compressionInfo,
		dedupInfo)
}

// createHasher creates a hasher based on the configured hash algorithm
func CreateHasher(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case constants.HashAlgorithmSHA256, "": // Default to SHA-256 if empty
		return sha256.New(), nil
	case constants.HashAlgorithmSHA512:
		return sha512.New(), nil
	case constants.HashAlgorithmSHA1:
		// #nosec G401
		return sha1.New(), nil
	case constants.HashAlgorithmBLAKE3:
		return blake3.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}
