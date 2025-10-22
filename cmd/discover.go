/*
Copyright © 2025 SubstantialCattle5 <nilaysharan.com>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover Sietch peers on your local network",
	Long: `Discover other Sietch vaults on your local network using mDNS.

This command creates a temporary libp2p node that broadcasts its presence and
listens for other Sietch vaults on the local network. When peers are discovered,
their information is displayed, including their peer ID and addresses.

With the --select flag, you can interactively choose which peers to pair with
for selective key exchange, providing fine-grained control over trust relationships.

Examples:
  sietch discover                  # Run discovery with default settings
  sietch discover --timeout 30     # Run discovery for 30 seconds
  sietch discover --continuous     # Run discovery until interrupted
  sietch discover --select         # Interactive peer selection for pairing
  sietch discover --port 9001      # Use a specific port for the libp2p node`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get command flags
		timeout, _ := cmd.Flags().GetInt("timeout")
		continuous, _ := cmd.Flags().GetBool("continuous")
		selectMode, _ := cmd.Flags().GetBool("select")
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

		fmt.Printf("🔍 Starting peer discovery with node ID: %s\n", host.ID().String())
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

		// Setup discovery
		discovery, peerChan, err := discover.SetupDiscovery(ctx, host)
		if err != nil {
			return err
		}
		defer func() { _ = discovery.Stop() }()

		// Run the discovery loop
		if selectMode {
			return runDiscoveryWithSelection(ctx, host, syncService, peerChan, timeout, continuous)
		}
		return discover.RunDiscoveryLoop(ctx, host, syncService, peerChan, timeout, continuous)
	},
}

// displayHostAddresses prints the addresses the host is listening on
func displayHostAddresses(h host.Host) {
	fmt.Println("Listening on:")
	for _, addr := range h.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, h.ID().String())
	}
}

func init() {
	rootCmd.AddCommand(discoverCmd)

	// Add command flags
	discoverCmd.Flags().IntP("timeout", "t", 60, "Discovery timeout in seconds (ignored with --continuous)")
	discoverCmd.Flags().BoolP("continuous", "c", false, "Run discovery continuously until interrupted")
	discoverCmd.Flags().BoolP("select", "s", false, "Interactive peer selection mode for pairing")
	discoverCmd.Flags().IntP("port", "p", 0, "Port to use for libp2p (0 for random port)")
	discoverCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	discoverCmd.Flags().StringP("vault-path", "V", "", "Path to the vault directory (defaults to current directory)")
}

// runDiscoveryWithSelection runs discovery and allows interactive peer selection for pairing
func runDiscoveryWithSelection(ctx context.Context, host host.Host, syncService *p2p.SyncService, peerChan <-chan peer.AddrInfo, timeout int, continuous bool) error {
	// Disable auto-trust for selective pairing
	syncService.SetAutoTrustAllPeers(false)

	// Collect discovered peers
	discoveredPeers := make([]peer.AddrInfo, 0)
	discoveredPeersMap := make(map[peer.ID]bool)

	var timeoutChan <-chan time.Time
	if !continuous {
		timeoutChan = time.After(time.Duration(timeout) * time.Second)
		fmt.Printf("   Discovery will run for %d seconds. Press Ctrl+C to stop earlier.\n\n", timeout)
	} else {
		fmt.Println("   Discovery will run until interrupted. Press Ctrl+C to stop.")
		fmt.Println()
	}

	fmt.Println("🔍 Discovering peers for selection...")

	// Discovery loop
	discoveryComplete := false
	for !discoveryComplete {
		select {
		case p, ok := <-peerChan:
			if !ok {
				discoveryComplete = true
				continue
			}

			if p.ID == host.ID() || discoveredPeersMap[p.ID] {
				continue
			}

			discoveredPeersMap[p.ID] = true
			discoveredPeers = append(discoveredPeers, p)

			fmt.Printf("✅ Discovered peer #%d\n", len(discoveredPeers))
			fmt.Printf("   ID: %s\n", p.ID.String())
			fmt.Println("   Addresses:")
			for _, addr := range p.Addrs {
				fmt.Printf("     - %s\n", addr.String())
			}
			fmt.Println()

		case <-timeoutChan:
			fmt.Printf("\n⌛ Discovery timeout reached after %d seconds.\n", timeout)
			discoveryComplete = true

		case <-ctx.Done():
			fmt.Println("\nDiscovery interrupted")
			discoveryComplete = true
		}
	}

	if len(discoveredPeers) == 0 {
		fmt.Println("No peers discovered. Make sure other Sietch vaults are running and discoverable.")
		return nil
	}

	// Let user select peers
	selectedPeers, err := ui.SelectPeersInteractively(discoveredPeers)
	if err != nil {
		return fmt.Errorf("peer selection failed: %v", err)
	}

	if len(selectedPeers) == 0 {
		fmt.Println("No peers selected for pairing.")
		return nil
	}

	// Set up pairing window (5 minutes default)
	windowDuration := 5 * time.Minute
	until := time.Now().Add(windowDuration)

	// Request pairing with selected peers
	for _, peerID := range selectedPeers {
		syncService.RequestPair(peerID, until)
	}

	fmt.Printf("\n⏰ Pairing window active for %v\n", windowDuration)
	fmt.Println("Waiting for mutual pairing...")

	// Wait for pairing to complete
	timeoutChan = time.After(windowDuration)
	pairedCount := 0

	for {
		select {
		case <-timeoutChan:
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
