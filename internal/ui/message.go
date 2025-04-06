package ui

import (
	"fmt"
	"strings"

	"github.com/substantialcattle5/sietch/internal/config"
)

func PrintSuccessMessage(config *config.VaultConfig, vaultID, vaultPath string) {
	// Create a visual separator
	separator := strings.Repeat("─", 50)

	// Success header with emoji
	fmt.Println("\n✅ Sietch Vault successfully initialized!")
	fmt.Println(separator)

	// Vault details section
	fmt.Println("📦 Vault Details:")
	fmt.Printf("  • Name:      %s\n", config.Name)
	fmt.Printf("  • ID:        %s\n", vaultID)
	fmt.Printf("  • Location:  %s\n", vaultPath)

	// Security details
	fmt.Println("\n🔒 Security:")
	fmt.Printf("  • Encryption: %s", config.Encryption.Type)
	if config.Encryption.PassphraseProtected {
		fmt.Print(" (passphrase protected)")
	}
	fmt.Println()

	// Storage configuration
	fmt.Println("\n💾 Storage:")
	fmt.Printf("  • Chunking:    %s (avg. %s MB)\n", config.Chunking.Strategy, config.Chunking.ChunkSize)
	fmt.Printf("  • Compression: %s\n", config.Chunking.HashAlgorithm)
	fmt.Printf("  • Manifest:    vault.yaml\n")

	// Metadata
	fmt.Println("\n📋 Metadata:")
	fmt.Printf("  • Author: %s\n", config.Metadata.Author)
	fmt.Printf("  • Tags:   %s\n", strings.Join(config.Metadata.Tags, ", "))

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
	fmt.Printf("  • Your vault configuration is stored at %s/vault.yaml\n", vaultPath)

	fmt.Println("\nThank you for using Sietch Vault! 🏜️")
}
