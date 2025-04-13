// cmd/sync.go
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/p2p"
	"github.com/substantialcattle5/sietch/util"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [peer-address]",
	Short: "Synchronize with another Sietch vault",
	Long: `Synchronize files with another Sietch vault over the network.

This command syncs your vault with another vault, either by auto-discovering
peers on the local network or by connecting to a specified peer address.

Examples:
  sietch sync                               # Auto-discover and sync with peers
  sietch sync /ip4/192.168.1.5/tcp/4001/p2p/QmPeerID  # Sync with a specific peer`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Find the vault root
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Create a libp2p host
		port, _ := cmd.Flags().GetInt("port")
		var opts []libp2p.Option
		if port > 0 {
			opts = append(opts, libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)))
		} else {
			opts = append(opts, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
		}

		host, err := libp2p.New(opts...)
		if err != nil {
			return fmt.Errorf("failed to create libp2p host: %v", err)
		}
		defer host.Close()

		fmt.Printf("🔌 Started Sietch node with ID: %s\n", host.ID().String())

		// Load the vault manager
		vaultMgr, err := config.NewManager(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault: %v", err)
		}

		// Create the sync service
		syncService, err := p2p.NewSyncService(host, vaultMgr)
		if err != nil {
			return fmt.Errorf("failed to create sync service: %v", err)
		}

		// Specific peer address provided
		if len(args) > 0 {
			peerAddr := args[0]
			fmt.Printf("🔄 Connecting to peer: %s\n", peerAddr)

			// Parse the multiaddress
			maddr, err := multiaddr.NewMultiaddr(peerAddr)
			if err != nil {
				return fmt.Errorf("invalid peer address: %v", err)
			}

			// Extract the peer ID from the multiaddress
			info, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				return fmt.Errorf("failed to parse peer info: %v", err)
			}

			// Connect to the peer
			if err := host.Connect(ctx, *info); err != nil {
				return fmt.Errorf("failed to connect to peer: %v", err)
			}

			fmt.Printf("✅ Connected to peer: %s\n", info.ID.String())
			fmt.Println("📝 Starting vault synchronization...")

			// Sync with the peer
			result, err := syncService.SyncWithPeer(ctx, info.ID)
			if err != nil {
				return fmt.Errorf("sync failed: %v", err)
			}

			// Display sync results
			displaySyncResults(result)
			return nil
		}

		// Auto-discovery mode
		fmt.Println("🔍 No peer specified, starting auto-discovery...")

		// Create the discovery factory
		factory := p2p.NewFactory()

		// Create and start mDNS discovery
		discovery, err := factory.CreateMDNS(host)
		if err != nil {
			return fmt.Errorf("failed to create mDNS discovery: %v", err)
		}

		if err := discovery.Start(ctx); err != nil {
			return fmt.Errorf("failed to start mDNS discovery: %v", err)
		}
		defer discovery.Stop()

		fmt.Println("📡 Searching for peers on local network...")

		// Set timeout for discovery
		timeout, _ := cmd.Flags().GetInt("timeout")
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer timeoutCancel()

		// Wait for peers
		select {
		case peerInfo := <-discovery.DiscoveredPeers():
			// Check if it's our own peer ID
			if peerInfo.ID == host.ID() {
				fmt.Println("🔄 Found our own peer, continuing discovery...")
				// Continue waiting for other peers
				select {
				case peerInfo = <-discovery.DiscoveredPeers():
					if peerInfo.ID == host.ID() {
						return fmt.Errorf("only found our own peer, no others on network")
					}
				case <-timeoutCtx.Done():
					return fmt.Errorf("discovery timed out after %d seconds", timeout)
				}
			}

			fmt.Printf("✅ Found peer: %s\n", peerInfo.ID.String())

			// Connect to the peer
			if err := host.Connect(ctx, peerInfo); err != nil {
				return fmt.Errorf("failed to connect to peer: %v", err)
			}

			fmt.Printf("🔄 Starting sync with peer: %s\n", peerInfo.ID.String())

			// Sync with the peer
			result, err := syncService.SyncWithPeer(ctx, peerInfo.ID)
			if err != nil {
				return fmt.Errorf("sync failed: %v", err)
			}

			// Display sync results
			displaySyncResults(result)

		case <-timeoutCtx.Done():
			return fmt.Errorf("discovery timed out after %d seconds, no peers found", timeout)
		}

		return nil
	},
}

// displaySyncResults shows the results of a sync operation
func displaySyncResults(result *p2p.SyncResult) {
	fmt.Println("\n✅ Synchronization complete!")
	fmt.Printf("   Files transferred:    %d\n", result.FileCount)
	fmt.Printf("   Chunks transferred:   %d\n", result.ChunksTransferred)
	fmt.Printf("   Chunks deduplicated:  %d\n", result.ChunksDeduplicated)
	fmt.Printf("   Data transferred:     %s\n", util.HumanReadableSize(result.BytesTransferred))
	fmt.Printf("   Duration:             %s\n", result.Duration.Round(time.Millisecond))
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Add command flags
	syncCmd.Flags().IntP("port", "p", 0, "Port to use for libp2p (0 for random port)")
	syncCmd.Flags().IntP("timeout", "t", 60, "Discovery timeout in seconds (for auto-discovery)")
}
