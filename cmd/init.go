package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/manifest"
)

var (
	vaultName string
	vaultPath string
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

}

func runInit() error {
	vaultID := uuid.New().String()

	absVaultPath, err := filepath.Abs(filepath.Join(vaultPath, vaultName))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create directory structure
	if err := fs.CreateVaultStructure(absVaultPath); err != nil {
		return err
	}

	configuration := config.BuildVaultConfig(vaultID, vaultName, absVaultPath)

	if err := manifest.WriteManifest(absVaultPath, configuration); err != nil {
		return err
	}
	return nil
}
