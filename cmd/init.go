/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
	"github.com/substantialcattle5/sietch/internal/ui"
)

var (
	vaultName string
	vaultPath string

	// Security key generation
	keyType       string
	usePassphrase bool
	keyFile       string

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

var initCmd = &cobra.Command{Use: "init",
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
		return runInit()
	}}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags with smart defaults
	initCmd.Flags().StringVar(&vaultName, "name", "my-sietch", "Name of the vault")
	initCmd.Flags().StringVar(&vaultPath, "path", ".", "Path to create the vault")

	// Encryption vars
	initCmd.Flags().StringVar(&keyType, "key-type", "aes", "Type of encryption key (aes, gpg, none)")
	initCmd.Flags().BoolVar(&usePassphrase, "passphrase", false, "Protect key with passphrase")
	initCmd.Flags().StringVar(&keyFile, "key-file", "", "Path to key file (for importing an existing key)")

	// AES specific parameters
	initCmd.Flags().StringVar(&aesMode, "aes-mode", "gcm", "AES encryption mode (gcm, cbc)")
	initCmd.Flags().BoolVar(&useScrypt, "use-scrypt", false, "Use scrypt for key derivation")
	initCmd.Flags().IntVar(&scryptN, "scrypt-n", 32768, "scrypt N parameter")
	initCmd.Flags().IntVar(&scryptR, "scrypt-r", 8, "scrypt r parameter")
	initCmd.Flags().IntVar(&scryptP, "scrypt-p", 1, "scrypt p parameter")

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

	// Other options
	initCmd.Flags().BoolVar(&interactiveMode, "interactive", false, "Use interactive mode")
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Force re-initialization of existing vault")
	initCmd.Flags().StringVar(&templateName, "template", "", "Use a predefined template structure")
	initCmd.Flags().StringVar(&configFile, "from-config", "", "Initialize from a configuration file")
}

func runInit() error {
	// If interactive mode is enabled, prompt for inputs
	interactiveVaultConfig, err := handleInteractiveMode()
	if err != nil {
		return err
	}

	// Set default author if not provided
	if author == "" {
		author = "sietch-user@example.com"
	}

	// If no tags are provided, set default tags
	if len(tags) == 0 {
		tags = []string{"research", "desert", "offline"}
	}

	absVaultPath, err := filepath.Abs(filepath.Join(vaultPath, vaultName))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if vault already exists by checking for .sietch directory
	sietchDir := filepath.Join(absVaultPath, ".sietch")
	if _, err := os.Stat(sietchDir); err == nil {
		if !forceInit {
			return fmt.Errorf("vault already exists at %s. Use --force to re-initialize (warning: this will destroy existing data)", absVaultPath)
		}
		// If force is true, we'll continue and overwrite
	}

	// Apply template configuration if specified
	if templateName != "" {
		fmt.Printf("Applying template: %s\n", templateName)
		// This would require implementing template functionality
	}

	// Load configuration from file if specified
	if configFile != "" {
		fmt.Printf("Loading configuration from: %s\n", configFile)
		// This would require implementing config loading functionality
	}

	vaultID := uuid.New().String()

	// Create directory structure
	if err := fs.CreateVaultStructure(absVaultPath); err != nil {
		return fmt.Errorf("failed to create vault structure: %w", err)
	}

	// Handle key generation or import
	keyPath := filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")
	var keyConfig *config.KeyConfig

	if keyFile != "" {
		// Import key from file
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file %s: %w", keyFile, err)
		}

		// Ensure directory exists
		keyDir := filepath.Dir(keyPath)
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			return fmt.Errorf("failed to create key directory: %w", err)
		}

		// Write key file
		if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
			return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
		}

		fmt.Printf("Imported key from %s\n", keyFile)
	} else {
		kdfValue := "pbkdf2"
		if useScrypt {
			kdfValue = "scrypt"
		}
		// Create encryption config
		encConfig := &config.VaultConfig{
			Encryption: config.EncryptionConfig{
				Type:                keyType,
				PassphraseProtected: usePassphrase,
				KeyFile:             keyFile != "",
				KeyFilePath:         keyFile,
				AESConfig: &config.AESConfig{
					Mode:    aesMode,
					KDF:     kdfValue,
					ScryptN: scryptN,
					ScryptR: scryptR,
					ScryptP: scryptP,
					PBKDF2I: 10000, // Default PBKDF2 iterations if not using scrypt
				},
			},
		}

		// Get passphrase if needed
		var passphrase string
		if usePassphrase {
			passphrasePrompt := promptui.Prompt{
				Label: "Enter encryption passphrase",
				Mask:  '*',
				Validate: func(input string) error {
					if len(input) < 8 {
						return fmt.Errorf("passphrase must be at least 8 characters")
					}
					return nil
				},
			}

			passphrase, err = passphrasePrompt.Run()
			if err != nil {
				return fmt.Errorf("failed to get passphrase: %w", err)
			}

			// Add confirmation prompt
			confirmPrompt := promptui.Prompt{
				Label: "Confirm passphrase",
				Mask:  '*',
				Validate: func(input string) error {
					if input != passphrase {
						return fmt.Errorf("passphrases do not match")
					}
					return nil
				},
			}

			_, err = confirmPrompt.Run()
			if err != nil {
				return fmt.Errorf("passphrase confirmation failed: %w", err)
			}
		}

		// Generate the key configuration
		keyConfig, err = keys.GenerateAESKey(encConfig, passphrase)
		if err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}

		// If we need to save the key to a file
		if encConfig.Encryption.KeyBackupPath != "" {
			fmt.Printf("Key backed up to: %s\n", encConfig.Encryption.KeyBackupPath)
		}
	}

	// If we didn't generate key config but have one from interactive mode
	if keyConfig == nil && keyType == "aes" && interactiveVaultConfig != nil && interactiveVaultConfig.Encryption.AESConfig != nil {
		keyConfig = &config.KeyConfig{
			AESConfig: interactiveVaultConfig.Encryption.AESConfig,
		}
	}

	// Build vault configuration
	configuration := config.BuildVaultConfig(
		vaultID,
		vaultName,
		author,
		keyType,
		keyPath,
		usePassphrase,
		chunkingStrategy,
		chunkSize,
		hashAlgorithm,
		compressionType,
		syncMode,
		tags,
		keyConfig, // Pass keyConfig as the last parameter
	)

	// Write configuration to manifest
	if err := manifest.WriteManifest(absVaultPath, configuration); err != nil {
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
	if keyType == "aes" && vaultConfig.Encryption.AESConfig != nil {
		// Set AES mode (GCM or CBC)
		aesMode = vaultConfig.Encryption.AESConfig.Mode

		// Handle KDF settings
		if vaultConfig.Encryption.AESConfig.KDF == "scrypt" {
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
