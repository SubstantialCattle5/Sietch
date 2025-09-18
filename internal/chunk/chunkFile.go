package chunk

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/compression"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/util"
)

func ChunkFile(filePath string, chunkSize int64, vaultRoot string, passphrase string) ([]config.ChunkRef, error) {
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

	return processFileChunks(file, chunkSize, *vaultConfig, chunksDir, passphrase)
}

func processFileChunks(file *os.File, chunkSize int64, vaultConfig config.VaultConfig, chunksDir string, passphrase string) ([]config.ChunkRef, error) {
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

		// Calculate chunk hash (pre-encryption)
		hasher := sha256.New()
		hasher.Write(buffer[:bytesRead])
		chunkHash := fmt.Sprintf("%x", hasher.Sum(nil))

		// Store original chunk data for processing
		originalChunkData := buffer[:bytesRead]

		// Apply compression if configured
		compressedData, err := compression.CompressData(originalChunkData, vaultConfig.Compression)
		if err != nil {
			return nil, fmt.Errorf("failed to compress chunk %d: %v", chunkCount, err)
		}

		// Create chunk reference
		chunkRef := config.ChunkRef{
			Hash:       chunkHash,
			Size:       int64(bytesRead),
			Index:      chunkCount - 1, // 0-based index
			Compressed: vaultConfig.Compression != "none",
		}

		// Use compressed data for further processing
		chunkDataToProcess := compressedData

		// Encrypt the chunk if encryption is enabled
		if vaultConfig.Encryption.Type != "" && vaultConfig.Encryption.Type != "none" {
			// Convert chunk data to string for encryption (use compressed data)
			chunkData := string(chunkDataToProcess)

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
				return nil, fmt.Errorf("failed to encrypt chunk %d: %v", chunkCount, encryptErr)
			}

			// Calculate hash of encrypted data for storage filename
			encHasher := sha256.New()
			encHasher.Write([]byte(encryptedData))
			encryptedHash := fmt.Sprintf("%x", encHasher.Sum(nil))

			// Update chunk reference with encryption info
			chunkRef.EncryptedHash = encryptedHash
			chunkRef.EncryptedSize = int64(len(encryptedData))

			// Save the encrypted chunk
			chunkPath := filepath.Join(chunksDir, encryptedHash)
			if err := os.WriteFile(chunkPath, []byte(encryptedData), constants.StandardFilePerms); err != nil {
				return nil, fmt.Errorf("failed to write encrypted chunk file: %v", err)
			}

			compressionInfo := ""
			if vaultConfig.Compression != "none" {
				compressionInfo = fmt.Sprintf(" (compressed with %s: %s -> %s)",
					vaultConfig.Compression,
					util.HumanReadableSize(int64(bytesRead)),
					util.HumanReadableSize(int64(len(chunkDataToProcess))))
			}

			fmt.Printf("Chunk %d: %s bytes, hash: %s (encrypted)%s\n",
				chunkCount,
				util.HumanReadableSize(int64(bytesRead)),
				chunkHash[:12],
				compressionInfo)
		} else {
			// If no encryption, save the compressed chunk
			chunkPath := filepath.Join(chunksDir, chunkHash)
			if err := os.WriteFile(chunkPath, chunkDataToProcess, constants.StandardFilePerms); err != nil {
				return nil, fmt.Errorf("failed to write chunk file: %v", err)
			}

			compressionInfo := ""
			if vaultConfig.Compression != "none" {
				compressionInfo = fmt.Sprintf(" (compressed with %s: %s -> %s)",
					vaultConfig.Compression,
					util.HumanReadableSize(int64(bytesRead)),
					util.HumanReadableSize(int64(len(chunkDataToProcess))))
			}

			fmt.Printf("Chunk %d: %s bytes, hash: %s%s\n",
				chunkCount,
				util.HumanReadableSize(int64(bytesRead)),
				chunkHash,
				compressionInfo)
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
