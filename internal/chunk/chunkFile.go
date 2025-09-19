package chunk

import (
	// #nosec G401

	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/substantialcattle5/sietch/internal/compression"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/util"
)

const (
	// Display constants
	HashDisplayLength = 12 // Length of hash to display in logs
)

func ChunkFile(filePath string, chunkSize int64, vaultRoot string, passphrase string) ([]config.ChunkRef, error) {
	// Validate input parameters
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be positive, got: %d", chunkSize)
	}

	file, err := fs.VerifyFileAndReturnFile(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Load Vault Configuration
	vaultConfig, err := config.LoadVaultConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %v", err)
	}

	// Ensure chunks directory exists
	chunksDir := fs.GetChunkDirectory(vaultRoot)
	if err := os.MkdirAll(chunksDir, constants.StandardDirPerms); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Check if vault requires passphrase but none was provided
	if vaultConfig.Encryption.Type == constants.EncryptionTypeAES && vaultConfig.Encryption.PassphraseProtected && passphrase == "" {
		return nil, fmt.Errorf("passphrase required for encrypted vault but not provided")
	}

	// Initialize deduplication manager
	dedupManager, err := deduplication.NewManager(vaultRoot, vaultConfig.Deduplication)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize deduplication manager: %v", err)
	}

	chunkRefs, err := processFileChunks(file, chunkSize, *vaultConfig, passphrase, dedupManager)
	if err != nil {
		return nil, err
	}

	// Save deduplication index after processing
	if err := dedupManager.Save(); err != nil {
		return nil, fmt.Errorf("failed to save deduplication index: %v", err)
	}

	return chunkRefs, nil
}

func processFileChunks(file *os.File, chunkSize int64, vaultConfig config.VaultConfig, passphrase string, dedupManager *deduplication.Manager) ([]config.ChunkRef, error) {
	// Create a buffer for reading chunks
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	totalBytes := int64(0)
	chunkRefs := []config.ChunkRef{}

	// Read the file in chunks
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading file: %v", err)
		}

		if bytesRead == 0 {
			// End of file
			break
		}

		chunkCount++
		totalBytes += int64(bytesRead)

		// Calculate chunk hash (pre-encryption) using configured algorithm
		hasher, err := CreateHasher(vaultConfig.Chunking.HashAlgorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to create hasher for chunk %d (algorithm: %s): %v", chunkCount, vaultConfig.Chunking.HashAlgorithm, err)
		}
		hasher.Write(buffer[:bytesRead])
		chunkHash := fmt.Sprintf("%x", hasher.Sum(nil))

		// Store original chunk data for processing
		originalChunkData := buffer[:bytesRead]

		// Apply compression if configured
		compressedData, err := compression.CompressData(originalChunkData, vaultConfig.Compression)
		if err != nil {
			return nil, fmt.Errorf("failed to compress chunk %d (size: %d bytes, algorithm: %s): %v", chunkCount, bytesRead, vaultConfig.Compression, err)
		}

		// Create chunk reference
		chunkRef := config.ChunkRef{
			Hash:       chunkHash,
			Size:       int64(bytesRead),
			Index:      chunkCount - 1, // Convert 1-based chunkCount to 0-based index
			Compressed: vaultConfig.Compression != "none",
		}

		// Use compressed data for further processing
		chunkDataToProcess := compressedData

		// Encrypt the chunk if encryption is enabled
		if vaultConfig.Encryption.Type != "" && vaultConfig.Encryption.Type != "none" {
			// Encode binary data to base64 string for safe encryption (use compressed data)
			chunkData := base64.StdEncoding.EncodeToString(chunkDataToProcess)

			var encryptedData string
			var encryptErr error

			// Choose encryption method based on passphrase protection
			if vaultConfig.Encryption.PassphraseProtected {
				encryptedData, encryptErr = encryption.EncryptDataWithPassphrase(
					chunkData,
					vaultConfig,
					passphrase,
				)
			} else {
				encryptedData, encryptErr = encryption.EncryptData(
					chunkData,
					vaultConfig,
				)
			}

			if encryptErr != nil {
				return nil, fmt.Errorf("failed to encrypt chunk %d (size: %d bytes, type: %s): %v", chunkCount, len(chunkDataToProcess), vaultConfig.Encryption.Type, encryptErr)
			}

			// Calculate hash of encrypted data for storage filename using configured algorithm
			encHasher, err := CreateHasher(vaultConfig.Chunking.HashAlgorithm)
			if err != nil {
				return nil, fmt.Errorf("failed to create encrypted hasher for chunk %d (algorithm: %s): %v", chunkCount, vaultConfig.Chunking.HashAlgorithm, err)
			}
			encHasher.Write([]byte(encryptedData))
			encryptedHash := fmt.Sprintf("%x", encHasher.Sum(nil))

			// Update chunk reference with encryption info
			chunkRef.EncryptedHash = encryptedHash
			chunkRef.EncryptedSize = int64(len(encryptedData))

			// Process chunk with deduplication manager
			updatedChunkRef, deduplicated, err := dedupManager.ProcessChunk(chunkRef, []byte(encryptedData), encryptedHash)
			if err != nil {
				return nil, fmt.Errorf("failed to process chunk %d with deduplication (encrypted, hash: %s): %v", chunkCount, encryptedHash[:HashDisplayLength], err)
			}
			chunkRef = updatedChunkRef

			// Display chunk information using helper function
			FormatChunkInfo(chunkCount, bytesRead, chunkHash, vaultConfig, chunkDataToProcess, deduplicated, true)
		} else {
			// If no encryption, process chunk with deduplication manager
			updatedChunkRef, deduplicated, err := dedupManager.ProcessChunk(chunkRef, chunkDataToProcess, chunkHash)
			if err != nil {
				return nil, fmt.Errorf("failed to process chunk %d with deduplication (unencrypted, hash: %s): %v", chunkCount, chunkHash[:HashDisplayLength], err)
			}
			chunkRef = updatedChunkRef

			// Display chunk information using helper function
			FormatChunkInfo(chunkCount, bytesRead, chunkHash, vaultConfig, chunkDataToProcess, deduplicated, false)
		}

		// Add the chunk reference to our list
		chunkRefs = append(chunkRefs, chunkRef)

		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Total chunks processed: %d\n", chunkCount)
	fmt.Printf("Total bytes processed: %s\n", util.HumanReadableSize(totalBytes))

	return chunkRefs, nil
}
