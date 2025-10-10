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

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/discover"
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
		vaultPath, _ := cmd.Flags().GetString("vault-path")
		allAddresses, _ := cmd.Flags().GetBool("all-addresses")

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
			discover.DisplayHostAddresses(host, allAddresses)
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
		return discover.RunDiscoveryLoop(ctx, host, syncService, peerChan, timeout, continuous, allAddresses)
	},
}



func init() {
	rootCmd.AddCommand(discoverCmd)

	// Add command flags
	discoverCmd.Flags().IntP("timeout", "t", 60, "Discovery timeout in seconds (ignored with --continuous)")
	discoverCmd.Flags().BoolP("continuous", "c", false, "Run discovery continuously until interrupted")
	discoverCmd.Flags().IntP("port", "p", 0, "Port to use for libp2p (0 for random port)")
	discoverCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	discoverCmd.Flags().StringP("vault-path", "V", "", "Path to the vault directory (defaults to current directory)")
	discoverCmd.Flags().Bool("all-addresses", false, "Show all network addresses including Docker, VPN, and virtual interfaces")
}
