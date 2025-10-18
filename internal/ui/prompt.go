package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/manifoldco/promptui"

	"github.com/substantialcattle5/sietch/internal/chunk"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption"
	"github.com/substantialcattle5/sietch/internal/vault"
)

// PromptForInputs guides the user through setting up a vault configuration
func PromptForInputs() (*config.VaultConfig, error) {
	configuration := &config.VaultConfig{}

	// Display welcome message
	fmt.Println("📦 Setting up your Sietch Vault")
	fmt.Println("===============================")
	fmt.Println("Let's configure your secure vault with the following steps:")
	fmt.Println()

	// Group 1: Basic Configuration
	fmt.Println("🔹 Basic Configuration")
	if err := vault.PromptBasicConfig(configuration); err != nil {
		return nil, err
	}

	// Group 2: Security Configuration
	fmt.Println("\n🔹 Security Configuration")
	if err := encryption.PromptSecurityConfig(configuration); err != nil {
		return nil, err
	}

	// Group 3: Chunking & Compression
	fmt.Println("\n🔹 Storage Configuration")
	if err := chunk.PromptStorageConfig(configuration); err != nil {
		return nil, err
	}

	// Group 4: Metadata
	fmt.Println("\n🔹 Metadata")
	if err := vault.PromptMetadataConfig(configuration); err != nil {
		return nil, err
	}

	// Display summary before confirmation
	displayConfigSummary(configuration)

	// Final confirmation
	confirmPrompt := promptui.Prompt{
		Label:     "Create vault with these settings",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := confirmPrompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return nil, errors.New("operation canceled")
		}
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	return configuration, nil
}

// displayConfigSummary shows a clean summary of the configuration
func displayConfigSummary(configuration *config.VaultConfig) {
	fmt.Println()
	fmt.Println("📋 Configuration Summary")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	// Basic Information
	fmt.Println("🏷️  Basic Information:")
	fmt.Printf("   • Vault Name: %s\n", configuration.Name)
	fmt.Printf("   • Author:     %s\n", configuration.Metadata.Author)

	tagsStr := strings.Join(configuration.Metadata.Tags, ", ")
	if tagsStr == "" {
		tagsStr = "none"
	}
	fmt.Printf("   • Tags:       %s\n", tagsStr)
	fmt.Println()

	// Security Configuration
	fmt.Println("🔐 Security:")
	encryptionDesc := getEncryptionDescription(configuration.Encryption)
	fmt.Printf("   • Encryption: %s\n", encryptionDesc)
	fmt.Println()

	// Storage Configuration
	fmt.Println("💾 Storage:")
	fmt.Printf("   • Chunking:    %s\n", configuration.Chunking.Strategy)
	fmt.Printf("   • Chunk Size:  %s\n", configuration.Chunking.ChunkSize)
	fmt.Printf("   • Hash Algo:   %s\n", configuration.Chunking.HashAlgorithm)

	compressionDesc := configuration.Compression
	if compressionDesc == "" {
		compressionDesc = "none"
	}
	fmt.Printf("   • Compression: %s\n", compressionDesc)

	fmt.Println()
	fmt.Println(strings.Repeat("=", 52))
}

// getEncryptionDescription returns a human-readable description of the encryption config
func getEncryptionDescription(enc config.EncryptionConfig) string {
	if enc.Type == "" || enc.Type == "none" {
		return "None ⚠️  (not recommended)"
	}

	desc := strings.ToUpper(enc.Type)
	if enc.PassphraseProtected {
		desc += " 🔒 (passphrase protected)"
	}

	// Add specific details for different encryption types
	switch enc.Type {
	case "aes":
		if enc.AESConfig != nil && enc.AESConfig.Mode != "" {
			desc += "-" + strings.ToUpper(enc.AESConfig.Mode)
		} else {
			desc += "-GCM" // default mode
		}
	case "gpg":
		if enc.GPGConfig != nil && enc.GPGConfig.KeyID != "" {
			keyID := enc.GPGConfig.KeyID
			if len(keyID) > 8 {
				keyID = keyID[:8]
			}
			desc += fmt.Sprintf(" (Key: %s)", keyID)
		}
	}

	return desc
}

// PeerSelectionItem represents a peer in the selection list
type PeerSelectionItem struct {
	PeerID    peer.ID
	Addresses []string
	Selected  bool
}

// SelectPeersInteractively allows users to select multiple peers from a list
func SelectPeersInteractively(peers []peer.AddrInfo) ([]peer.ID, error) {
	if len(peers) == 0 {
		fmt.Println("No peers available for selection.")
		return []peer.ID{}, nil
	}

	// Convert to selection items
	items := make([]PeerSelectionItem, len(peers))
	for i, p := range peers {
		addresses := make([]string, len(p.Addrs))
		for j, addr := range p.Addrs {
			addresses[j] = addr.String()
		}
		items[i] = PeerSelectionItem{
			PeerID:    p.ID,
			Addresses: addresses,
			Selected:  false,
		}
	}

	fmt.Printf("\n🔍 Found %d peer(s) on the network:\n", len(peers))
	fmt.Println(strings.Repeat("=", 50))

	// Display peer list
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item.PeerID.String())
		for _, addr := range item.Addresses {
			fmt.Printf("   └─ %s\n", addr)
		}
		fmt.Println()
	}

	// Get selection from user
	selectionPrompt := promptui.Prompt{
		Label: fmt.Sprintf("Select peers to pair with (1-%d, comma-separated, or 'all' for all peers)", len(peers)),
		Validate: func(input string) error {
			if input == "" {
				return errors.New("please select at least one peer")
			}
			return nil
		},
	}

	input, err := selectionPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("selection prompt failed: %w", err)
	}

	// Parse selection
	var selectedPeers []peer.ID
	if strings.ToLower(input) == "all" {
		// Select all peers
		for _, item := range items {
			selectedPeers = append(selectedPeers, item.PeerID)
		}
	} else {
		// Parse comma-separated indices
		indices := strings.Split(input, ",")
		for _, indexStr := range indices {
			indexStr = strings.TrimSpace(indexStr)
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, fmt.Errorf("invalid selection '%s': %w", indexStr, err)
			}
			if index < 1 || index > len(peers) {
				return nil, fmt.Errorf("selection %d is out of range (1-%d)", index, len(peers))
			}
			selectedPeers = append(selectedPeers, items[index-1].PeerID)
		}
	}

	// Remove duplicates
	uniquePeers := make([]peer.ID, 0, len(selectedPeers))
	seen := make(map[peer.ID]bool)
	for _, peerID := range selectedPeers {
		if !seen[peerID] {
			uniquePeers = append(uniquePeers, peerID)
			seen[peerID] = true
		}
	}

	fmt.Printf("✅ Selected %d peer(s) for pairing:\n", len(uniquePeers))
	for _, peerID := range uniquePeers {
		fmt.Printf("   • %s\n", peerID.String())
	}

	return uniquePeers, nil
}

// PromptForPairingWindow asks user for pairing window duration
func PromptForPairingWindow() (int, error) {
	windowPrompt := promptui.Prompt{
		Label:   "Pairing window duration in minutes",
		Default: "5",
		Validate: func(input string) error {
			var minutes int
			if _, err := fmt.Sscanf(input, "%d", &minutes); err != nil {
				return errors.New("please enter a valid number of minutes")
			}
			if minutes < 1 || minutes > 60 {
				return errors.New("pairing window must be between 1 and 60 minutes")
			}
			return nil
		},
	}

	input, err := windowPrompt.Run()
	if err != nil {
		return 0, fmt.Errorf("pairing window prompt failed: %w", err)
	}

	var minutes int
	fmt.Sscanf(input, "%d", &minutes)
	return minutes, nil
}

// PromptForIncomingPeers asks user which peers to allow for incoming pairing
func PromptForIncomingPeers(peers []peer.AddrInfo) ([]peer.ID, error) {
	if len(peers) == 0 {
		fmt.Println("No peers available for incoming pairing permission.")
		return []peer.ID{}, nil
	}

	fmt.Printf("\n🔐 Grant incoming pairing permission to %d peer(s):\n", len(peers))
	fmt.Println(strings.Repeat("=", 50))

	// Display peer list
	for i, p := range peers {
		fmt.Printf("%d. %s\n", i+1, p.ID.String())
		for _, addr := range p.Addrs {
			fmt.Printf("   └─ %s\n", addr.String())
		}
		fmt.Println()
	}

	// Get selection from user
	selectionPrompt := promptui.Prompt{
		Label: fmt.Sprintf("Allow incoming pairing from peers (1-%d, comma-separated, or 'all' for all peers)", len(peers)),
		Validate: func(input string) error {
			if input == "" {
				return errors.New("please select at least one peer")
			}
			return nil
		},
	}

	input, err := selectionPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("incoming peer selection prompt failed: %w", err)
	}

	// Parse selection (same logic as SelectPeersInteractively)
	var selectedPeers []peer.ID
	if strings.ToLower(input) == "all" {
		// Select all peers
		for _, p := range peers {
			selectedPeers = append(selectedPeers, p.ID)
		}
	} else {
		// Parse comma-separated indices
		indices := strings.Split(input, ",")
		for _, indexStr := range indices {
			indexStr = strings.TrimSpace(indexStr)
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, fmt.Errorf("invalid selection '%s': %w", indexStr, err)
			}
			if index < 1 || index > len(peers) {
				return nil, fmt.Errorf("selection %d is out of range (1-%d)", index, len(peers))
			}
			selectedPeers = append(selectedPeers, peers[index-1].ID)
		}
	}

	// Remove duplicates
	uniquePeers := make([]peer.ID, 0, len(selectedPeers))
	seen := make(map[peer.ID]bool)
	for _, peerID := range selectedPeers {
		if !seen[peerID] {
			uniquePeers = append(uniquePeers, peerID)
			seen[peerID] = true
		}
	}

	fmt.Printf("✅ Granted incoming pairing permission to %d peer(s):\n", len(uniquePeers))
	for _, peerID := range uniquePeers {
		fmt.Printf("   • %s\n", peerID.String())
	}

	return uniquePeers, nil
}
