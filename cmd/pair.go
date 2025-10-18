/*
Copyright © 2025 SubstantialCattle5 <nilaysharan.com>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/discover"
	"github.com/substantialcattle5/sietch/internal/p2p"
	"github.com/substantialcattle5/sietch/internal/ui"
)

// pairCmd represents the pair command
var pairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Pair with specific peers for selective key exchange",
	Long: `Establish selective trust relationships with other Sietch vaults.

This command allows you to choose which peers you want to exchange keys with,
providing fine-grained control over your vault's trust relationships. Key exchange
only occurs when both parties have mutually selected each other.

Examples:
  sietch pair --select                    # Interactive peer selection
  sietch pair --allow-from <peerID>       # Allow specific peer to pair with you
  sietch pair --allow-all --window 10m    # Allow all peers for 10 minutes
  sietch pair --select --window 5m        # Select peers with 5-minute window`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get command flags
		selectMode, _ := cmd.Flags().GetBool("select")
		allowFrom, _ := cmd.Flags().GetString("allow-from")
		allowAll, _ := cmd.Flags().GetBool("allow-all")
		window, _ := cmd.Flags().GetString("window")
		port, _ := cmd.Flags().GetInt("port")
		verbose, _ := cmd.Flags().GetBool("verbose")
		vaultPath, _ := cmd.Flags().GetString("vault-path")

		// If no vault path specified, use current directory
		if vaultPath == "" {
			var err error
			vaultPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}
		}

		// Parse window duration
		windowDuration, err := parseWindowDuration(window)
		if err != nil {
			return fmt.Errorf("invalid window duration: %v", err)
		}

		// Validate flags
		if !selectMode && allowFrom == "" && !allowAll {
			return fmt.Errorf("must specify one of: --select, --allow-from, or --allow-all")
		}

		// Create a context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle interrupts gracefully
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-signalChan
			fmt.Println("\nReceived interrupt signal, shutting down...")
			cancel()
		}()

		// Create a libp2p host
		host, err := p2p.CreateLibp2pHost(port)
		if err != nil {
			return fmt.Errorf("failed to create libp2p host: %v", err)
		}
		defer host.Close()

		fmt.Printf("🔗 Starting pairing mode with node ID: %s\n", host.ID().String())
		if verbose {
			displayHostAddresses(host)
		}

		// Create a vault manager
		vaultMgr, err := config.NewManager(vaultPath)
		if err != nil {
			return fmt.Errorf("failed to create vault manager: %v", err)
		}

		// Get vault config
		vaultConfig, err := vaultMgr.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		// Create sync service (with or without RSA)
		syncService, err := discover.CreateSyncService(host, vaultMgr, vaultConfig, vaultPath, verbose)
		if err != nil {
			return fmt.Errorf("failed to create sync service: %v", err)
		}

		// Disable auto-trust for selective pairing
		syncService.SetAutoTrustAllPeers(false)

		// Set pairing window
		syncService.SetPairingWindow(windowDuration)

		// Setup discovery
		discovery, peerChan, err := discover.SetupDiscovery(ctx, host)
		if err != nil {
			return err
		}
		defer func() { _ = discovery.Stop() }()

		fmt.Printf("📡 Discovering peers for %v...\n", windowDuration)

		// Collect discovered peers
		discoveredPeers := make([]peer.AddrInfo, 0)
		timeout := time.After(windowDuration)

	discoveryLoop:
		for {
			select {
			case p, ok := <-peerChan:
				if !ok {
					break discoveryLoop
				}

				if p.ID == host.ID() {
					continue
				}

				// Check if we already have this peer
				alreadyFound := false
				for _, existing := range discoveredPeers {
					if existing.ID == p.ID {
						alreadyFound = true
						break
					}
				}

				if !alreadyFound {
					discoveredPeers = append(discoveredPeers, p)
					fmt.Printf("✅ Discovered peer: %s\n", p.ID.String())
					if verbose {
						for _, addr := range p.Addrs {
							fmt.Printf("   └─ %s\n", addr.String())
						}
					}
				}

			case <-timeout:
				fmt.Printf("\n⌛ Discovery timeout reached after %v\n", windowDuration)
				break discoveryLoop

			case <-ctx.Done():
				fmt.Println("\nDiscovery interrupted")
				break discoveryLoop
			}
		}

		if len(discoveredPeers) == 0 {
			fmt.Println("No peers discovered. Make sure other Sietch vaults are running and discoverable.")
			return nil
		}

		// Handle different pairing modes
		if selectMode {
			return handleSelectMode(ctx, host, syncService, discoveredPeers, windowDuration)
		} else if allowFrom != "" {
			return handleAllowFromMode(ctx, host, syncService, discoveredPeers, allowFrom, windowDuration)
		} else if allowAll {
			return handleAllowAllMode(ctx, host, syncService, discoveredPeers, windowDuration)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pairCmd)

	// Add flags
	pairCmd.Flags().Bool("select", false, "Interactive peer selection mode")
	pairCmd.Flags().String("allow-from", "", "Comma-separated list of peer IDs to allow for incoming pairing")
	pairCmd.Flags().Bool("allow-all", false, "Allow all discovered peers for incoming pairing")
	pairCmd.Flags().String("window", "5m", "Pairing window duration (e.g., 5m, 10m, 1h)")
	pairCmd.Flags().Int("port", 0, "Port for libp2p host (0 for random)")
	pairCmd.Flags().Bool("verbose", false, "Enable verbose output")
	pairCmd.Flags().String("vault-path", "", "Path to vault directory")
}

// parseWindowDuration parses a duration string like "5m", "10m", "1h"
func parseWindowDuration(window string) (time.Duration, error) {
	if window == "" {
		return 5 * time.Minute, nil
	}

	// Handle common formats
	switch window {
	case "1m", "1min":
		return 1 * time.Minute, nil
	case "5m", "5min":
		return 5 * time.Minute, nil
	case "10m", "10min":
		return 10 * time.Minute, nil
	case "30m", "30min":
		return 30 * time.Minute, nil
	case "1h", "1hour":
		return 1 * time.Hour, nil
	}

	// Try to parse as Go duration
	duration, err := time.ParseDuration(window)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s': %w", window, err)
	}

	// Limit to reasonable range
	if duration < 1*time.Minute || duration > 1*time.Hour {
		return 0, fmt.Errorf("duration must be between 1 minute and 1 hour")
	}

	return duration, nil
}

// handleSelectMode handles interactive peer selection
func handleSelectMode(ctx context.Context, host host.Host, syncService *p2p.SyncService, peers []peer.AddrInfo, windowDuration time.Duration) error {
	fmt.Println("\n🎯 Interactive peer selection mode")

	// Let user select peers
	selectedPeers, err := ui.SelectPeersInteractively(peers)
	if err != nil {
		return fmt.Errorf("peer selection failed: %v", err)
	}

	if len(selectedPeers) == 0 {
		fmt.Println("No peers selected for pairing.")
		return nil
	}

	// Set up pairing window
	until := time.Now().Add(windowDuration)

	// Request pairing with selected peers
	for _, peerID := range selectedPeers {
		syncService.RequestPair(peerID, until)
	}

	fmt.Printf("\n⏰ Pairing window active for %v\n", windowDuration)
	fmt.Println("Waiting for mutual pairing...")

	// Wait for pairing to complete
	return waitForPairing(ctx, host, syncService, selectedPeers, windowDuration)
}

// handleAllowFromMode handles specific peer allowlist
func handleAllowFromMode(ctx context.Context, host host.Host, syncService *p2p.SyncService, peers []peer.AddrInfo, allowFrom string, windowDuration time.Duration) error {
	fmt.Println("\n🔐 Allowlist mode")

	// Parse peer IDs
	peerIDStrings := strings.Split(allowFrom, ",")
	allowedPeers := make([]peer.ID, 0, len(peerIDStrings))

	for _, peerIDStr := range peerIDStrings {
		peerIDStr = strings.TrimSpace(peerIDStr)
		peerID, err := peer.Decode(peerIDStr)
		if err != nil {
			return fmt.Errorf("invalid peer ID '%s': %v", peerIDStr, err)
		}
		allowedPeers = append(allowedPeers, peerID)
	}

	// Set up pairing window
	until := time.Now().Add(windowDuration)

	// Allow incoming pairing from specified peers
	for _, peerID := range allowedPeers {
		syncService.AllowIncomingPair(peerID, until)
	}

	fmt.Printf("✅ Allowed incoming pairing from %d peer(s) for %v\n", len(allowedPeers), windowDuration)

	// Display our node ID for others to select
	fmt.Printf("\n📢 Your node ID: %s\n", host.ID().String())
	fmt.Println("Share this ID with others so they can select you for pairing.")
	fmt.Println("Waiting for incoming pairing requests...")

	// Wait for pairing to complete
	return waitForIncomingPairing(ctx, host, syncService, allowedPeers, windowDuration)
}

// handleAllowAllMode handles allowing all discovered peers
func handleAllowAllMode(ctx context.Context, host host.Host, syncService *p2p.SyncService, peers []peer.AddrInfo, windowDuration time.Duration) error {
	fmt.Println("\n🌐 Allow all mode")

	// Set up pairing window
	until := time.Now().Add(windowDuration)

	// Allow incoming pairing from all discovered peers
	for _, peer := range peers {
		syncService.AllowIncomingPair(peer.ID, until)
	}

	fmt.Printf("✅ Allowed incoming pairing from all %d discovered peer(s) for %v\n", len(peers), windowDuration)

	// Display our node ID for others to select
	fmt.Printf("\n📢 Your node ID: %s\n", host.ID().String())
	fmt.Println("Share this ID with others so they can select you for pairing.")
	fmt.Println("Waiting for incoming pairing requests...")

	// Wait for pairing to complete
	return waitForIncomingPairing(ctx, host, syncService, nil, windowDuration)
}

// waitForPairing waits for pairing to complete with selected peers
func waitForPairing(ctx context.Context, host host.Host, syncService *p2p.SyncService, selectedPeers []peer.ID, windowDuration time.Duration) error {
	timeout := time.After(windowDuration)
	pairedCount := 0

	for {
		select {
		case <-timeout:
			fmt.Printf("\n⌛ Pairing window expired after %v\n", windowDuration)
			fmt.Printf("Successfully paired with %d peer(s)\n", pairedCount)
			return nil

		case <-ctx.Done():
			fmt.Printf("\nPairing interrupted. Successfully paired with %d peer(s)\n", pairedCount)
			return nil

		default:
			// Check if any selected peers have been successfully paired
			for _, peerID := range selectedPeers {
				if syncService.HasPeer(peerID) {
					pairedCount++
					fmt.Printf("✅ Successfully paired with peer: %s\n", peerID.String())

					// Remove from selected list to avoid counting twice
					for i, p := range selectedPeers {
						if p == peerID {
							selectedPeers = append(selectedPeers[:i], selectedPeers[i+1:]...)
							break
						}
					}
				}
			}

			// If all peers are paired, we're done
			if len(selectedPeers) == 0 {
				fmt.Printf("\n🎉 All selected peers successfully paired!\n")
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// waitForIncomingPairing waits for incoming pairing requests
func waitForIncomingPairing(ctx context.Context, host host.Host, syncService *p2p.SyncService, allowedPeers []peer.ID, windowDuration time.Duration) error {
	timeout := time.After(windowDuration)
	pairedCount := 0

	for {
		select {
		case <-timeout:
			fmt.Printf("\n⌛ Pairing window expired after %v\n", windowDuration)
			fmt.Printf("Successfully paired with %d peer(s)\n", pairedCount)
			return nil

		case <-ctx.Done():
			fmt.Printf("\nPairing interrupted. Successfully paired with %d peer(s)\n", pairedCount)
			return nil

		default:
			// Check if any allowed peers have been successfully paired
			if allowedPeers != nil {
				for _, peerID := range allowedPeers {
					if syncService.HasPeer(peerID) {
						pairedCount++
						fmt.Printf("✅ Successfully paired with peer: %s\n", peerID.String())

						// Remove from allowed list to avoid counting twice
						for i, p := range allowedPeers {
							if p == peerID {
								allowedPeers = append(allowedPeers[:i], allowedPeers[i+1:]...)
								break
							}
						}
					}
				}
			} else {
				// In allow-all mode, count all trusted peers
				// This is a simplified check - in practice you'd want to track which ones were added during this session
				pairedCount = len(syncService.TrustedPeers())
			}

			time.Sleep(1 * time.Second)
		}
	}
}
