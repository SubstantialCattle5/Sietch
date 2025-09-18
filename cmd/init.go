/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/internal/ui"
	"github.com/substantialcattle5/sietch/internal/validation"
	"github.com/substantialcattle5/sietch/internal/vault"
)

var (
	vaultName string
	vaultPath string

	// Security key generation
	keyType         string
	usePassphrase   bool
	keyFile         string
	passphraseValue string

	// aes specific keys
	aesMode   string
	scryptN   int
	scryptR   int
	scryptP   int
	useScrypt bool

	// Chunking configuration
	chunkingStrategy string
	chunkSize        string
	hashAlgorithm    string

	// Compression
	compressionType string

	// Sync
	syncMode string

	// Metadata
	author string
	tags   []string

	// Other options
	interactiveMode bool
	forceInit       bool
	templateName    string
	configFile      string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Sietch vault",
	Long: `Initialize a new Sietch vault with secure encryption and configurable options.
This creates the necessary directory structure and configuration files for your vault.

Examples:
  # Quickstart vault with defaults
  sietch init
  
  # Named vault with AES key + passphrase
  sietch init --name "desert-cache" --key-type aes --passphrase
  
  # AES with custom scrypt parameters
  sietch init --key-type aes --passphrase --use-scrypt --scrypt-n 32768 --scrypt-r 8 --scrypt-p 1
  
  # AES with key file
  sietch init --key-type aes --key-file path/to/key.bin

  # Custom chunking and GPG encryption
  sietch init --chunking-strategy cdc --chunk-size 2MB --key-type gpg
  
  # Use config file from template or backup
  sietch init --from-config my-old-vault.yaml
  
  # Use predefined template
  sietch init --template photo-vault
  
  # Force re-initialization of an existing vault
  sietch init --force`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags with smart defaults
	initCmd.Flags().StringVar(&vaultName, "name", "my-sietch", "Name of the vault")
	initCmd.Flags().StringVar(&vaultPath, "path", ".", "Path to create the vault")

	// Encryption vars
	initCmd.Flags().StringVar(&keyType, "key-type", "aes", "Type of encryption key (aes, gpg, none)")
	initCmd.Flags().BoolVar(&usePassphrase, "passphrase", false, "Protect key with passphrase")
	initCmd.Flags().StringVar(&keyFile, "key-file", "", "Path to key file (for importing an existing key)")
	initCmd.Flags().StringVar(&passphraseValue, "passphrase-value", "", "Passphrase for encryption (NOT RECOMMENDED: passphrase will be visible in command history)")

	// AES specific parameters
	initCmd.Flags().StringVar(&aesMode, "aes-mode", "gcm", "AES encryption mode (gcm, cbc)")
	initCmd.Flags().BoolVar(&useScrypt, "use-scrypt", false, "Use scrypt for key derivation")
	initCmd.Flags().IntVar(&scryptN, "scrypt-n", constants.DefaultScryptN, "scrypt N parameter")
	initCmd.Flags().IntVar(&scryptR, "scrypt-r", constants.DefaultScryptR, "scrypt r parameter")
	initCmd.Flags().IntVar(&scryptP, "scrypt-p", constants.DefaultScryptP, "scrypt p parameter")

	// Chunking vars
	initCmd.Flags().StringVar(&chunkingStrategy, "chunking-strategy", "fixed", "Strategy for chunking (fixed, cdc)")
	initCmd.Flags().StringVar(&chunkSize, "chunk-size", "4MB", "Size of chunks")
	initCmd.Flags().StringVar(&hashAlgorithm, "hash", "sha256", "Hash algorithm (sha256, blake3)")

	// Compression vars
	initCmd.Flags().StringVar(&compressionType, "compression", "none", "Compression type (none, gzip, zstd)")

	// Sync vars
	initCmd.Flags().StringVar(&syncMode, "sync-mode", "manual", "Synchronization mode (manual, auto)")

	// Metadata vars
	initCmd.Flags().StringVar(&author, "author", "", "Author metadata")
	initCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Tags for vault")

	// RSA Keys
	initCmd.Flags().Int("rsa-bits", constants.DefaultRSAKeySize, "Bit size for the RSA key pair (min 2048, recommended 4096)")

	// Other options
	initCmd.Flags().BoolVar(&interactiveMode, "interactive", false, "Use interactive mode")
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Force re-initialization of existing vault")
	initCmd.Flags().StringVar(&templateName, "template", "", "Use a predefined template structure")
	initCmd.Flags().StringVar(&configFile, "from-config", "", "Initialize from a configuration file")
}

func runInit(cmd *cobra.Command) error {
	// Handle interactive mode first
	interactiveVaultConfig, err := handleInteractiveMode()
	if err != nil {
		return err
	}

	// Validate and prepare inputs
	authorValidated, tagsValidated, err := validation.ValidateAndPrepareInputs(author, tags, templateName, configFile)
	if err != nil {
		return err
	}
	// Update the original variables with validated values
	author = authorValidated
	tags = tagsValidated

	// Prepare vault path and check for existing vault
	absVaultPath, err := vault.PrepareVaultPath(vaultPath, vaultName, forceInit)
	if err != nil {
		return err
	}

	// Create directory structure
	if err := fs.CreateVaultStructure(absVaultPath); err != nil {
		return fmt.Errorf("failed to create vault structure: %w", err)
	}

	// Handle key generation or import
	var keyConfig *config.KeyConfig

	// If we have a GPG config from interactive mode, use it directly
	if interactiveVaultConfig != nil && keyType == constants.EncryptionTypeGPG && interactiveVaultConfig.Encryption.GPGConfig != nil {
		keyConfig = &config.KeyConfig{
			GPGConfig: interactiveVaultConfig.Encryption.GPGConfig,
			KeyHash:   interactiveVaultConfig.Encryption.KeyHash,
		}
	} else {
		// Otherwise, generate or import a new key
		keyParams := validation.KeyGenParams{
			KeyType:          keyType,
			UsePassphrase:    usePassphrase,
			KeyFile:          keyFile,
			AESMode:          aesMode,
			UseScrypt:        useScrypt,
			ScryptN:          scryptN,
			ScryptR:          scryptR,
			ScryptP:          scryptP,
			PBKDF2Iterations: constants.DefaultPBKDF2Iters, // Default PBKDF2 iterations
		}

		var err error
		keyConfig, err = validation.HandleKeyGeneration(cmd, absVaultPath, keyParams)
		if err != nil {
			// Clean up on error
			cleanupOnError(absVaultPath)
			return fmt.Errorf("key generation failed: %w", err)
		}

		// If we didn't generate key config but have one from interactive mode (for AES)
		if keyConfig == nil && interactiveVaultConfig != nil {
			if keyType == constants.EncryptionTypeAES && interactiveVaultConfig.Encryption.AESConfig != nil {
				keyConfig = &config.KeyConfig{
					AESConfig: interactiveVaultConfig.Encryption.AESConfig,
				}
			}
		}
	}

	// // Print the key config to verify it contains the key
	// // TODO: think we should remove this
	// if keyConfig != nil && keyConfig.AESConfig != nil {
	// 	fmt.Println("\nKey Configuration:")
	// 	fmt.Printf("  Key exists: %v\n", keyConfig.AESConfig.Key != "")
	// 	// Print first few chars of the key if it exists (for debugging)
	// 	if keyConfig.AESConfig.Key != "" {
	// 		keyLen := len(keyConfig.AESConfig.Key)
	// 		if keyLen > 10 {
	// 			fmt.Printf("  Key (first 10 chars): %s...\n", keyConfig.AESConfig.Key[:10])
	// 		} else {
	// 			fmt.Printf("  Key: %s\n", keyConfig.AESConfig.Key)
	// 		}
	// 	}
	// }

	// Generate vault ID
	vaultID := uuid.New().String()

	// Create the key path for storing the key file
	keyPath := filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")

	// Write the key to file if it exists
	if keyType == constants.EncryptionTypeAES && keyConfig != nil && keyConfig.AESConfig != nil && keyConfig.AESConfig.Key != "" {
		// Decode the base64-encoded key
		keyMaterial, err := base64.StdEncoding.DecodeString(keyConfig.AESConfig.Key)
		if err != nil {
			cleanupOnError(absVaultPath)
			return fmt.Errorf("failed to decode key: %w", err)
		}

		// Create directory structure for the key if it doesn't exist
		keyDir := filepath.Dir(keyPath)
		if err := os.MkdirAll(keyDir, constants.SecureDirPerms); err != nil {
			cleanupOnError(absVaultPath)
			return fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
		}

		// Write the key with secure permissions (only owner can read/write)
		if err := os.WriteFile(keyPath, keyMaterial, constants.SecureFilePerms); err != nil {
			cleanupOnError(absVaultPath)
			return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
		}

		fmt.Printf("Encryption key stored at: %s\n", keyPath)
	}

	// Build vault configuration
	configuration := config.BuildVaultConfig(
		vaultID,
		vaultName,
		authorValidated,
		keyType,
		keyPath,
		usePassphrase,
		chunkingStrategy,
		chunkSize,
		hashAlgorithm,
		compressionType,
		syncMode,
		tags,
		keyConfig,
	)

	// Initialize RSA config if not present
	if configuration.Sync.RSA == nil {
		configuration.Sync.RSA = &config.RSAConfig{
			KeySize:      constants.DefaultRSAKeySize,
			TrustedPeers: []config.TrustedPeer{},
		}
	}

	// Get RSA key size from flags
	rsaBits, err := cmd.Flags().GetInt("rsa-bits")
	if err == nil && rsaBits >= constants.MinRSAKeySize {
		configuration.Sync.RSA.KeySize = rsaBits
	}

	// Generate RSA key pair for sync
	err = keys.GenerateRSAKeyPair(absVaultPath, &configuration)
	if err != nil {
		cleanupOnError(absVaultPath)
		return fmt.Errorf("failed to generate RSA keys for sync: %w", err)
	}

	// Print the final configuration to verify it has the key
	fmt.Println("\nFinal Vault Configuration:")
	if configuration.Encryption.AESConfig != nil {
		fmt.Printf("  AES Key exists: %v\n", configuration.Encryption.AESConfig.Key != "")
	}

	// Write configuration to manifest
	if err := manifest.WriteManifest(absVaultPath, configuration); err != nil {
		cleanupOnError(absVaultPath)
		return fmt.Errorf("failed to write vault manifest: %w", err)
	}

	// Print success message
	ui.PrintSuccessMessage(&configuration, vaultID, absVaultPath)

	return nil
}

func handleInteractiveMode() (*config.VaultConfig, error) {
	if !interactiveMode {
		return nil, nil
	}

	vaultConfig, err := ui.PromptForInputs()
	if err != nil {
		return nil, fmt.Errorf("interactive input failed: %w", err)
	}
	// Update variables with values from vaultConfig
	vaultName = vaultConfig.Name
	keyType = vaultConfig.Encryption.Type
	usePassphrase = vaultConfig.Encryption.PassphraseProtected

	// Handle AES-specific encryption configuration
	if keyType == constants.EncryptionTypeAES && vaultConfig.Encryption.AESConfig != nil {
		// Set AES mode (GCM or CBC)
		aesMode = vaultConfig.Encryption.AESConfig.Mode

		// Handle KDF settings
		if vaultConfig.Encryption.AESConfig.KDF == constants.KDFScrypt {
			useScrypt = true
			scryptN = vaultConfig.Encryption.AESConfig.ScryptN
			scryptR = vaultConfig.Encryption.AESConfig.ScryptR
			scryptP = vaultConfig.Encryption.AESConfig.ScryptP
		} else {
			// PBKDF2 settings would be handled here
			useScrypt = false
		}

		// Handle key file settings
		if vaultConfig.Encryption.KeyFile {
			keyFile = vaultConfig.Encryption.KeyFilePath
		}
	}

	// Handle chunking configuration
	chunkingStrategy = vaultConfig.Chunking.Strategy
	chunkSize = vaultConfig.Chunking.ChunkSize
	hashAlgorithm = vaultConfig.Chunking.HashAlgorithm

	// Handle other configuration
	compressionType = vaultConfig.Compression
	syncMode = vaultConfig.Sync.Mode
	author = vaultConfig.Metadata.Author
	tags = vaultConfig.Metadata.Tags

	return vaultConfig, nil
}

func cleanupOnError(absVaultPath string) {
	// Attempt to clean up partially created vault on error
	_ = os.RemoveAll(absVaultPath)
}
