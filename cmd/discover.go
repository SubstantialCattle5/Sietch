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

	"github.com/libp2p/go-libp2p"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/p2p"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover Sietch peers on your local network",
	Long: `Discover other Sietch vaults on your local network using mDNS.

This command creates a temporary libp2p node that broadcasts its presence and
listens for other Sietch vaults on the local network. When peers are discovered,
their information is displayed, including their peer ID and addresses.

Example:
  sietch discover                  # Run discovery with default settings
  sietch discover --timeout 30     # Run discovery for 30 seconds
  sietch discover --continuous     # Run discovery until interrupted
  sietch discover --port 9001      # Use a specific port for the libp2p node`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get command flags
		timeout, _ := cmd.Flags().GetInt("timeout")
		continuous, _ := cmd.Flags().GetBool("continuous")
		port, _ := cmd.Flags().GetInt("port")
		verbose, _ := cmd.Flags().GetBool("verbose")

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

		// Configure libp2p host
		var opts []libp2p.Option
		if port > 0 {
			opts = append(opts, libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)))
		} else {
			opts = append(opts, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
		}

		// Create a libp2p host
		host, err := libp2p.New(opts...)
		if err != nil {
			return fmt.Errorf("failed to create libp2p host: %v", err)
		}
		defer host.Close()

		fmt.Printf("🔍 Starting peer discovery with node ID: %s\n", host.ID().String())
		if verbose {
			fmt.Println("Listening on:")
			for _, addr := range host.Addrs() {
				fmt.Printf("  %s/p2p/%s\n", addr, host.ID().String())
			}
		}

		// Create the discovery factory
		factory := p2p.NewFactory()

		// Create and start the mDNS discovery service
		discovery, err := factory.CreateMDNS(host)
		if err != nil {
			return fmt.Errorf("failed to create mDNS discovery service: %v", err)
		}

		if err := discovery.Start(ctx); err != nil {
			return fmt.Errorf("failed to start mDNS discovery: %v", err)
		}
		defer discovery.Stop()

		fmt.Println("📡 Scanning local network for Sietch vaults...")
		fmt.Println("   (Peers will appear as they're discovered)")
		fmt.Println()

		// Set up timeouts
		var timeoutChan <-chan time.Time
		if !continuous {
			timeoutChan = time.After(time.Duration(timeout) * time.Second)
			fmt.Printf("   Discovery will run for %d seconds. Press Ctrl+C to stop earlier.\n\n", timeout)
		} else {
			fmt.Println("   Discovery will run until interrupted. Press Ctrl+C to stop.")
			fmt.Println()
		}

		// Track discovered peers to avoid duplicates
		discoveredPeers := make(map[string]bool)
		peerCount := 0

		// Listen for discovered peers
		peerChan := discovery.DiscoveredPeers()
		for {
			select {
			case peer, ok := <-peerChan:
				if !ok {
					// Channel closed
					return nil
				}

				// Skip if this is our own peer ID or already discovered
				if peer.ID == host.ID() || discoveredPeers[peer.ID.String()] {
					continue
				}

				// Mark as discovered
				discoveredPeers[peer.ID.String()] = true
				peerCount++

				// Display peer information
				fmt.Printf("✅ Discovered peer #%d\n", peerCount)
				fmt.Printf("   ID: %s\n", peer.ID.String())
				fmt.Println("   Addresses:")
				for _, addr := range peer.Addrs {
					fmt.Printf("     - %s\n", addr.String())
				}

				// Try connecting to the peer if in verbose mode
				if verbose {
					fmt.Printf("   Attempting connection... ")
					connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
					if err := host.Connect(connectCtx, peer); err != nil {
						fmt.Printf("failed: %v\n", err)
					} else {
						fmt.Printf("success!\n")
					}
					connectCancel()
				}
				fmt.Println()

			case <-timeoutChan:
				// Timeout reached
				fmt.Printf("\n⌛ Discovery timeout reached after %d seconds.\n", timeout)
				if peerCount == 0 {
					fmt.Println("   No Sietch vaults were discovered on the local network.")
				} else {
					fmt.Printf("   Discovered %d Sietch vault(s) on the local network.\n", peerCount)
				}
				return nil

			case <-ctx.Done():
				// Context cancelled (interrupted)
				if peerCount == 0 {
					fmt.Println("\nNo Sietch vaults were discovered on the local network.")
				} else {
					fmt.Printf("\nDiscovered %d Sietch vault(s) on the local network.\n", peerCount)
				}
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)

	// Add command flags
	discoverCmd.Flags().IntP("timeout", "t", 60, "Discovery timeout in seconds (ignored with --continuous)")
	discoverCmd.Flags().BoolP("continuous", "c", false, "Run discovery continuously until interrupted")
	discoverCmd.Flags().IntP("port", "p", 0, "Port to use for libp2p (0 for random port)")
	discoverCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
}
