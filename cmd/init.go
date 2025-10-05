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
	"github.com/spf13/pflag"

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

	// Deduplication
	enableDeduplication bool
	dedupStrategy       string
	dedupMinChunkSize   string
	dedupMaxChunkSize   string
	dedupGCThreshold    int

	// Other options
	interactiveMode bool
	forceInit       bool
	templateName    string
	configFile      string
)

func shortHelp(cmd *cobra.Command) {
	fmt.Printf("Usage: %s\n\n", cmd.UseLine())
	fmt.Printf("%s\n\n", cmd.Short)
	fmt.Println("Run 'sietch init --help' for detailed examples and options.")
	fmt.Println(`
	Examples:

	# Quickstart vault with defaults and interactive mode
	sietch init --interactive
		
	# Quickstart vault with defaults
	sietch init --name "my-vault"


	# Named vault with AES key + passphrase
	sietch init --name "desert-cache" --key-type aes --passphrase
		
	# AES with custom scrypt parameters
	sietch init --key-type aes --passphrase --use-scrypt --scrypt-n 32768 --scrypt-r 8 --scrypt-p 1

	# ChaCha20 encryption with passphrase
	sietch init --key-type chacha20 --passphrase
  `)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Sietch vault",
	Long: `Initialize a new Sietch vault with secure encryption and configurable options.
This creates the necessary directory structure and configuration files for your vault.

Examples:
  # Show help and available options
  sietch init --help
  
  # Quickstart vault with defaults
  sietch init --name "my-vault"
  
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
	initCmd.Flags().StringVar(&keyType, "key-type", "aes", "Type of encryption key (aes, chacha20, gpg, none)")
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

	// Deduplication options
	initCmd.Flags().BoolVar(&enableDeduplication, "enable-dedup", true, "Enable deduplication (default: true)")
	initCmd.Flags().StringVar(&dedupStrategy, "dedup-strategy", "content", "Deduplication strategy (content)")
	initCmd.Flags().StringVar(&dedupMinChunkSize, "dedup-min-size", "1KB", "Minimum chunk size for deduplication")
	initCmd.Flags().StringVar(&dedupMaxChunkSize, "dedup-max-size", "64MB", "Maximum chunk size for deduplication")
	initCmd.Flags().IntVar(&dedupGCThreshold, "dedup-gc-threshold", 1000, "Unreferenced chunk count before GC suggestion")

	// Other options
	initCmd.Flags().BoolVar(&interactiveMode, "interactive", false, "Use interactive mode")
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Force re-initialization of existing vault")
	initCmd.Flags().StringVar(&templateName, "template", "", "Use a predefined template structure")
	initCmd.Flags().StringVar(&configFile, "from-config", "", "Initialize from a configuration file")
}

func runInit(cmd *cobra.Command) error {

	// Check if any flags were provided by the user
	// If no flags were provided, show shorter version (just usage and short description)
	hasChanged := false
	cmd.Flags().Visit(func(f *pflag.Flag) {
		hasChanged = true
	})

	if !hasChanged {
		// Show shorter help - just usage and short description
		shortHelp(cmd)
		return nil
	}

	// Check if --help was explicitly provided
	// If --help is provided, show the full help with examples
	if cmd.Flags().Changed("help") {
		return cmd.Help()
	}

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

	// If we have a config from interactive mode, use it directly
	if interactiveVaultConfig != nil && keyType == constants.EncryptionTypeGPG && interactiveVaultConfig.Encryption.GPGConfig != nil {
		keyConfig = &config.KeyConfig{
			GPGConfig: interactiveVaultConfig.Encryption.GPGConfig,
			KeyHash:   interactiveVaultConfig.Encryption.KeyHash,
		}
	} else if interactiveVaultConfig != nil && keyType == constants.EncryptionTypeAES && interactiveVaultConfig.Encryption.AESConfig != nil {
		// Use AES config from interactive mode (key already generated during interactive config)
		keyConfig = &config.KeyConfig{
			AESConfig: interactiveVaultConfig.Encryption.AESConfig,
			KeyHash:   interactiveVaultConfig.Encryption.KeyHash,
			Salt:      interactiveVaultConfig.Encryption.AESConfig.Salt,
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

	}

	// Generate vault ID
	vaultID := uuid.New().String()

	// Create the key path for storing the key file (for AES and ChaCha20 encryption)
	var keyPath string
	if keyType == constants.EncryptionTypeAES || keyType == constants.EncryptionTypeChaCha20 {
		keyPath = filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")
	}

	// Write the key to file if it exists
	if keyType == constants.EncryptionTypeAES && keyConfig != nil && keyConfig.AESConfig != nil && keyConfig.AESConfig.Key != "" {
		// Decode the base64-encoded AES key
		keyMaterial, err := base64.StdEncoding.DecodeString(keyConfig.AESConfig.Key)
		if err != nil {
			cleanupOnError(absVaultPath)
			return fmt.Errorf("failed to decode AES key: %w", err)
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
	} else if keyType == constants.EncryptionTypeChaCha20 && keyConfig != nil && keyConfig.ChaChaConfig != nil && keyConfig.ChaChaConfig.Key != "" {
		// Note: ChaCha20 key generation already writes the key to file in chachakey.GenerateChaCha20Key
		// So we don't need to write it again here, but we print confirmation
		fmt.Printf("Encryption key stored at: %s\n", keyPath)
	}

	// Build vault configuration
	configuration := config.BuildVaultConfigWithDeduplication(
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
		// Deduplication parameters
		enableDeduplication,
		dedupStrategy,
		dedupMinChunkSize,
		dedupMaxChunkSize,
		dedupGCThreshold,
		true, // index enabled
		// Default automatic GC settings
		true, "1h", 1000, true, ".sietch/logs/gc.log", 5000, "",
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

	// Handle ChaCha20-specific encryption configuration
	if keyType == constants.EncryptionTypeChaCha20 && vaultConfig.Encryption.ChaChaConfig != nil {
		// ChaCha20 uses scrypt by default, but we can set the parameters if specified
		if vaultConfig.Encryption.ChaChaConfig.KDF == constants.KDFScrypt {
			useScrypt = true
			scryptN = vaultConfig.Encryption.ChaChaConfig.ScryptN
			scryptR = vaultConfig.Encryption.ChaChaConfig.ScryptR
			scryptP = vaultConfig.Encryption.ChaChaConfig.ScryptP
		} else {
			// Default to scrypt for ChaCha20
			useScrypt = true
			scryptN = constants.DefaultScryptN
			scryptR = constants.DefaultScryptR
			scryptP = constants.DefaultScryptP
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

	// Handle deduplication configuration
	enableDeduplication = vaultConfig.Deduplication.Enabled
	dedupStrategy = vaultConfig.Deduplication.Strategy
	dedupMinChunkSize = vaultConfig.Deduplication.MinChunkSize
	dedupMaxChunkSize = vaultConfig.Deduplication.MaxChunkSize
	dedupGCThreshold = vaultConfig.Deduplication.GCThreshold

	return vaultConfig, nil
}

func cleanupOnError(absVaultPath string) {
	// Attempt to clean up partially created vault on error
	_ = os.RemoveAll(absVaultPath)
}
