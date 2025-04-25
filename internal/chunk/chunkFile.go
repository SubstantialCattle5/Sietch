package chunk

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/util"
)

func ChunkFile(filePath string, chunkSize int64, vaultRoot string) ([]config.ChunkRef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found at %s", filePath)
		}
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Load vault configuration to access encryption settings
	vaultConfig, err := config.LoadVaultConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %v", err)
	}

	// Ensure chunks directory exists
	chunksDir := fs.GetChunkDirectory(vaultRoot)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Create a buffer for reading chunks
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	totalBytes := int64(0)
	chunkRefs := []config.ChunkRef{}

	// Prepare passphrase if key is passphrase protected
	var passphrase string
	if vaultConfig.Encryption.PassphraseProtected {
		// In a real implementation, you would securely prompt for the passphrase
		// This is a placeholder - you should replace with secure passphrase handling
		passphrase = os.Getenv("SIETCH_PASSPHRASE")
		if passphrase == "" {
			return nil, fmt.Errorf("passphrase required for encrypted vault but not provided")
		}
	}

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

			// Use encryption with vault config
			encryptedData, err := encryption.AesEncryption(
				chunkData,
				vaultRoot,
			)

			if err != nil {
				return nil, fmt.Errorf("failed to encrypt chunk %d: %v", chunkCount, err)
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
			if err := os.WriteFile(chunkPath, []byte(encryptedData), 0644); err != nil {
				return nil, fmt.Errorf("failed to write encrypted chunk file: %v", err)
			}
		} else {
			// If no encryption, save the raw chunk
			chunkPath := filepath.Join(chunksDir, chunkHash)
			if err := os.WriteFile(chunkPath, buffer[:bytesRead], 0644); err != nil {
				return nil, fmt.Errorf("failed to write chunk file: %v", err)
			}
		}

		fmt.Printf("Chunk %d: %s bytes, hash: %s\n",
			chunkCount,
			util.HumanReadableSize(int64(bytesRead)),
			chunkHash,
		)

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

// ChunkFileWithPassphrase chunks a file and encrypts the chunks using the vault's encryption key
// The passphrase is used to decrypt the encryption key if the vault is passphrase protected
func ChunkFileWithPassphrase(filePath string, chunkSize int64, vaultRoot string, passphrase string) ([]config.ChunkRef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found at %s", filePath)
		}
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Load vault configuration to access encryption settings
	vaultConfig, err := config.LoadVaultConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %v", err)
	}

	// Ensure chunks directory exists
	chunksDir := fs.GetChunkDirectory(vaultRoot)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Create a buffer for reading chunks
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	totalBytes := int64(0)
	chunkRefs := []config.ChunkRef{}

	// Check if vault requires passphrase but none was provided
	if vaultConfig.Encryption.Type == "aes" && vaultConfig.Encryption.PassphraseProtected && passphrase == "" {
		return nil, fmt.Errorf("passphrase required for encrypted vault but not provided")
	}

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

			// Encrypt the chunk using the vault's encryption settings and the provided passphrase
			encryptedData, err := encryption.AesEncryptWithPassphrase(
				chunkData,
				vaultRoot,
				passphrase,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt chunk %d: %v", chunkCount, err)
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
			if err := os.WriteFile(chunkPath, []byte(encryptedData), 0644); err != nil {
				return nil, fmt.Errorf("failed to write encrypted chunk file: %v", err)
			}

			fmt.Printf("Chunk %d: %s bytes, hash: %s (encrypted)\n",
				chunkCount,
				util.HumanReadableSize(int64(bytesRead)),
				chunkHash[:12])
		} else {
			// If no encryption, save the raw chunk
			chunkPath := filepath.Join(chunksDir, chunkHash)
			if err := os.WriteFile(chunkPath, buffer[:bytesRead], 0644); err != nil {
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
