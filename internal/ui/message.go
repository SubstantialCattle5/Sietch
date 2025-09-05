package ui

import (
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
)

// PrintSuccessMessage displays a formatted success message after vault initialization
func PrintSuccessMessage(cfg *config.VaultConfig, vaultID, vaultPath string) {
	// Create a visual separator
	separator := strings.Repeat("─", 50)

	// Success header with emoji
	fmt.Println("\n✅ Sietch Vault successfully initialized!")
	fmt.Println(separator)

	// Vault details section
	fmt.Println("📦 Vault Details:")
	fmt.Printf("  • Name:      %s\n", cfg.Name)
	fmt.Printf("  • ID:        %s\n", vaultID)
	fmt.Printf("  • Location:  %s\n", vaultPath)

	// Security details
	fmt.Println("\n🔒 Security:")
	fmt.Printf("  • Encryption: %s", cfg.Encryption.Type)
	if cfg.Encryption.PassphraseProtected {
		fmt.Print(" (passphrase protected)")
	}
	fmt.Println()

	// Storage configuration
	fmt.Println("\n💾 Storage:")
	fmt.Printf("  • Chunking:    %s (size: %s)\n", cfg.Chunking.Strategy, cfg.Chunking.ChunkSize)
	fmt.Printf("  • Hash:        %s\n", cfg.Chunking.HashAlgorithm)
	fmt.Printf("  • Compression: %s\n", cfg.Compression)

	// Metadata
	fmt.Println("\n📋 Metadata:")
	fmt.Printf("  • Author: %s\n", cfg.Metadata.Author)
	fmt.Printf("  • Tags:   %s\n", strings.Join(cfg.Metadata.Tags, ", "))

	// Next steps and commands
	fmt.Println("\n" + separator)
	fmt.Println("🚀 Next Steps:")

	// Add files command with example
	fmt.Println("\n1️⃣ Add files to your vault:")
	fmt.Println("   sietch add path/to/file.txt path/to/directory")
	fmt.Println("   sietch add --recursive path/to/directory")

	// List vault contents
	fmt.Println("\n2️⃣ View vault contents:")
	fmt.Println("   sietch list")
	fmt.Println("   sietch status")

	// Sync commands
	fmt.Println("\n3️⃣ Sync with peers:")
	fmt.Println("   sietch sync --peer 192.168.1.100")
	fmt.Println("   sietch sync --discover  # find peers on local network")

	// Tips section
	fmt.Println("\n💡 Tips:")
	fmt.Println("  • Run 'sietch help' for a list of all commands")
	fmt.Println("  • Use 'sietch config' to view or modify vault settings")
	fmt.Printf("  • Your vault configuration is stored at %s/.sietch/vault.yaml\n", vaultPath)

	fmt.Println("\nThank you for using Sietch Vault! 🏜️")
}
