package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
)

var (
	vaultName string
	vaultPath string

	// Security key generation
	keyType       string
	usePassphrase bool

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
  
  # Custom chunking and GPG encryption
  sietch init --chunking-strategy cdc --chunk-size 2MB --key-type gpg
   `,

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

	// Chunking vars
	initCmd.Flags().StringVar(&chunkingStrategy, "chunking-strategy", "fixed", "Strategy for chunking (fixed, cdc)")
	initCmd.Flags().StringVar(&chunkSize, "chunk-size", "4MB", "Size of chunks")
	initCmd.Flags().StringVar(&hashAlgorithm, "hash", "sha256", "Hash algorithm (sha256, blake3)")

	// Compression vars
	initCmd.Flags().StringVar(&compressionType, "compression", "none", "Compression type (none, gzip, zstd)")

	// Sync vars
	initCmd.Flags().StringVar(&syncMode, "sync-mode", "manual", "Synchronization mode (manual, auto)")

	// Metadata vars
	initCmd.Flags().StringVar(&author, "author", "nilay@dune.net", "Author metadata")
	initCmd.Flags().StringSliceVar(&tags, "tags", []string{"research", "desert", "offline"}, "Tags for vault")

	// Other options
	initCmd.Flags().BoolVar(&interactiveMode, "interactive", false, "Use interactive mode")
}

func runInit() error {
	// If interactive mode is enabled, prompt for inputs
	if interactiveMode {
		if err := promptForInputs(); err != nil {
			return err
		}
	}

	vaultID := uuid.New().String()

	absVaultPath, err := filepath.Abs(filepath.Join(vaultPath, vaultName))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create directory structure
	if err := fs.CreateVaultStructure(absVaultPath); err != nil {
		return fmt.Errorf("failed to create vault structure: %w", err)
	}

	// Create encryption key
	keyPath := filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")
	if err := encryption.GenerateKey(keyType, keyPath, usePassphrase); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
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
	)

	// Write configuration to manifest
	if err := manifest.WriteManifest(absVaultPath, configuration); err != nil {
		return fmt.Errorf("failed to write vault manifest: %w", err)
	}

	// Print success message
	printSuccessMessage(vaultID, absVaultPath)

	return nil
}

// Interactive mode prompt handler
func promptForInputs() error {
	// In a real implementation, this would use a library like promptui
	// to create an interactive CLI experience
	fmt.Println("Interactive mode would prompt for:")
	fmt.Println("- Vault name")
	fmt.Println("- Key type (aes/gpg/none)")
	fmt.Println("- Passphrase protection")
	fmt.Println("- Chunking strategy and size")
	fmt.Println("- Compression settings")

	return nil
}

// Print success message after initialization
func printSuccessMessage(vaultID, vaultPath string) {
	fmt.Println("\nSietchVault initialized!")
	fmt.Printf("Vault ID: %s\n", vaultID)
	fmt.Printf("Location: %s\n", vaultPath)
	fmt.Printf("Encryption: %s", keyType)
	if usePassphrase {
		fmt.Print(" (passphrase protected)")
	}
	fmt.Println()
	fmt.Printf("Manifest: vault.yaml\n")

	// Additional instructions for the user
	fmt.Println("\nYou can now add files to your vault with:")
	fmt.Printf("  sietch add <files>\n")
	fmt.Println("\nOr sync your vault with another peer:")
	fmt.Printf("  sietch sync --peer <ip-address>\n")
}
