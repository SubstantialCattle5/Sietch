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
	"github.com/substantialcattle5/sietch/internal/scaffold"
	"github.com/substantialcattle5/sietch/internal/validation"
	"github.com/substantialcattle5/sietch/internal/vault"
)

func runScaffold(templateName, name, path string, force bool) error {
	// Ensure config directories exist
	if err := scaffold.EnsureConfigDirectories(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %v", err)
	}

	// Ensure default templates are available
	if err := scaffold.EnsureDefaultTemplates(); err != nil {
		return fmt.Errorf("failed to ensure default templates: %v", err)
	}

	// Load and validate the template
	template, err := scaffold.ValidateTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to validate template: %v", err)
	}

	fmt.Printf("Loading template: %s\n", template.Name)
	fmt.Printf("Description: %s\n", template.Description)

	// Use template name as vault name if not provided
	if name == "" {
		name = template.Name
	}

	// Use current directory if path not provided
	if path == "" {
		path = "."
	}

	// Prepare vault path and check for existing vault
	absVaultPath, err := vault.PrepareVaultPath(path, name, force)
	if err != nil {
		return err
	}

	// Create basic vault structure
	if err := fs.CreateVaultStructure(absVaultPath); err != nil {
		return fmt.Errorf("failed to create vault structure: %w", err)
	}

	// Generate encryption key using AES (default for templates)
	keyParams := validation.KeyGenParams{
		KeyType:          constants.EncryptionTypeAES,
		UsePassphrase:    false, // Default no passphrase for scaffolded vaults
		KeyFile:          "",
		AESMode:          constants.AESModeGCM,
		UseScrypt:        true,
		ScryptN:          constants.DefaultScryptN,
		ScryptR:          constants.DefaultScryptR,
		ScryptP:          constants.DefaultScryptP,
		PBKDF2Iterations: constants.DefaultPBKDF2Iters,
	}

	keyConfig, err := validation.HandleKeyGeneration(nil, absVaultPath, keyParams)
	if err != nil {
		scaffoldCleanupOnError(absVaultPath)
		return fmt.Errorf("key generation failed: %w", err)
	}

	// Generate vault ID
	vaultID := uuid.New().String()

	// Create the key path for storing the key file
	keyPath := filepath.Join(absVaultPath, ".sietch", "keys", "secret.key")

	// Write the key to file
	if keyConfig != nil && keyConfig.AESConfig != nil && keyConfig.AESConfig.Key != "" {
		// Decode the base64-encoded key
		keyMaterial, err := base64.StdEncoding.DecodeString(keyConfig.AESConfig.Key)
		if err != nil {
			scaffoldCleanupOnError(absVaultPath)
			return fmt.Errorf("failed to decode key: %w", err)
		}

		// Create directory structure for the key if it doesn't exist
		keyDir := filepath.Dir(keyPath)
		if err := os.MkdirAll(keyDir, constants.SecureDirPerms); err != nil {
			scaffoldCleanupOnError(absVaultPath)
			return fmt.Errorf("failed to create key directory %s: %w", keyDir, err)
		}

		// Write the key with secure permissions (only owner can read/write)
		if err := os.WriteFile(keyPath, keyMaterial, constants.SecureFilePerms); err != nil {
			scaffoldCleanupOnError(absVaultPath)
			return fmt.Errorf("failed to write key to %s: %w", keyPath, err)
		}

		fmt.Printf("Encryption key stored at: %s\n", keyPath)
	}

	// Build vault configuration using template settings
	cfg := &template.Config
	configuration := config.BuildVaultConfigWithDeduplication(
		vaultID,
		name,
		"", // Author will be prompted or use default
		constants.EncryptionTypeAES,
		keyPath,
		false, // No passphrase protection for scaffolded vaults
		cfg.ChunkingStrategy,
		cfg.ChunkSize,
		cfg.HashAlgorithm,
		cfg.Compression,
		cfg.SyncMode,
		template.Tags, // Use template tags
		keyConfig,
		// Deduplication parameters from template
		cfg.EnableDedup,
		cfg.DedupStrategy,
		cfg.DedupMinSize,
		cfg.DedupMaxSize,
		cfg.DedupGCThreshold,
		cfg.DedupIndexEnabled,
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

	// Generate RSA key pair for sync
	err = keys.GenerateRSAKeyPair(absVaultPath, &configuration)
	if err != nil {
		scaffoldCleanupOnError(absVaultPath)
		return fmt.Errorf("failed to generate RSA keys for sync: %w", err)
	}

	// Write configuration to manifest
	if err := manifest.WriteManifest(absVaultPath, configuration); err != nil {
		scaffoldCleanupOnError(absVaultPath)
		return fmt.Errorf("failed to write vault manifest: %w", err)
	}

	// Print success message
	fmt.Printf("\n✅ Successfully scaffolded '%s' vault at: %s\n", template.Name, absVaultPath)
	fmt.Printf("📝 Template: %s (v%s)\n", template.Name, template.Version)
	fmt.Printf("🔐 Encryption: AES-256-GCM\n")
	fmt.Printf("📦 Chunking: %s (%s chunks)\n", cfg.ChunkingStrategy, cfg.ChunkSize)
	if cfg.EnableDedup {
		fmt.Printf("♻️  Deduplication: Enabled (%s strategy)\n", cfg.DedupStrategy)
	}
	fmt.Printf("🗜️  Compression: %s\n", cfg.Compression)
	fmt.Printf("\nYour vault is ready to use! Add files with: sietch add <files>\n")

	return nil
}

func scaffoldCleanupOnError(absVaultPath string) {
	// Attempt to clean up partially created vault on error
	_ = os.RemoveAll(absVaultPath)
}

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Scaffold a new Sietch vault",
	Long: `Scaffold a new Sietch vault with secure encryption and configurable options.
This creates the necessary directory structure and configuration files for your vault
using pre-configured templates optimized for different use cases.

Available Templates:
  Photos & Media:
    photoVault     - Photo storage with strong dedup and high compression
    videoVault     - Video storage with large chunks and light compression
    audioLibrary   - Audio/podcast storage with balanced settings

  Documents & Knowledge:
    documentsVault - Office/PDF documents with aggressive compression
    codeVault      - Code repositories with fine-grained deduplication
    reporterVault  - Journalism/sensitive documents with manual sync

  Backups & Archives:
    systemBackup   - System backups optimized for performance
    coldArchive    - Long-term archival with maximum compression

Examples:
  List all available templates:
    sietch scaffold --list

  Create a vault from a template:
    sietch scaffold --template photoVault
    sietch scaffold --template videoVault --name "My Movies"
    sietch scaffold --template documentsVault --name "Work Docs" --path ~/Documents
    sietch scaffold --template codeVault --name "Projects" --path ~/Code --force

  Learn more about templates:
    See ~/.config/sietch/templates/README.md for detailed comparison`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if user wants to list templates
		list, _ := cmd.Flags().GetBool("list")
		if list {
			return scaffold.ListTemplates()
		}

		// Get flag values
		template, _ := cmd.Flags().GetString("template")
		if template == "" {
			return fmt.Errorf("template is required. Use --list to see available templates")
		}

		name, _ := cmd.Flags().GetString("name")
		path, _ := cmd.Flags().GetString("path")
		force, _ := cmd.Flags().GetBool("force")

		return runScaffold(template, name, path, force)
	},
}

func init() {
	rootCmd.AddCommand(scaffoldCmd)

	// Add required flags
	scaffoldCmd.Flags().StringP("template", "t", "", "Template to use for scaffolding (required)")
	scaffoldCmd.Flags().StringP("name", "n", "", "Name for the vault (optional)")
	scaffoldCmd.Flags().StringP("path", "p", "", "Path where to create the vault (optional)")
	scaffoldCmd.Flags().BoolP("force", "f", false, "Force creation even if directory exists")
	scaffoldCmd.Flags().BoolP("list", "l", false, "List available templates")

}
