package chunk

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

		// Create chunk reference
		chunkRef := config.ChunkRef{
			Hash:  chunkHash,
			Size:  int64(bytesRead),
			Index: chunkCount - 1, // 0-based index
		}

		// Encrypt the chunk if encryption is enabled
		if vaultConfig.Encryption.Type != "" && vaultConfig.Encryption.Type != "none" {
			// Convert chunk data to string for encryption
			chunkData := string(buffer[:bytesRead])

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

			fmt.Printf("Chunk %d: %s bytes, hash: %s (encrypted)\n",
				chunkCount,
				util.HumanReadableSize(int64(bytesRead)),
				chunkHash[:12])
		} else {
			// If no encryption, save the raw chunk
			chunkPath := filepath.Join(chunksDir, chunkHash)
			if err := os.WriteFile(chunkPath, buffer[:bytesRead], constants.StandardFilePerms); err != nil {
				return nil, fmt.Errorf("failed to write chunk file: %v", err)
			}

			fmt.Printf("Chunk %d: %s bytes, hash: %s\n",
				chunkCount,
				util.HumanReadableSize(int64(bytesRead)),
				chunkHash)
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
