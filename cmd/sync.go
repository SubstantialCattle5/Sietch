/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package cmd

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/p2p"
	"github.com/substantialcattle5/sietch/util"

	"github.com/substantialcattle5/sietch/internal/history"
	"github.com/google/uuid"
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

		// Load vault configuration
		vaultCfg, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault config: %v", err)
		}

		// Load RSA keys for secure communication
		privateKey, publicKey, err := loadRSAKeys(vaultRoot, vaultCfg)
		if err != nil {
			return fmt.Errorf("failed to load RSA keys: %v", err)
		}

		// Convert RSA private key to libp2p format
		libp2pPrivKey, err := rsaToLibp2pPrivateKey(privateKey)
		if err != nil {
			return fmt.Errorf("failed to convert RSA key to libp2p format: %v", err)
		}

		// Create a libp2p host with our identity key
		port, _ := cmd.Flags().GetInt("port")
		var opts []libp2p.Option

		// Use our RSA key as the node identity
		opts = append(opts, libp2p.Identity(libp2pPrivKey))

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

		// Print our listen addresses
		fmt.Println("📡 Listening on:")
		for _, addr := range host.Addrs() {
			fmt.Printf("   %s/p2p/%s\n", addr.String(), host.ID().String())
		}

		// Load the vault manager
		vaultMgr, err := config.NewManager(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault: %v", err)
		}

		// Create the sync service with RSA key information
		syncService, err := p2p.NewSecureSyncService(host, vaultMgr, privateKey, publicKey, vaultCfg.Sync.RSA)
		if err != nil {
			return fmt.Errorf("failed to create sync service: %v", err)
		}

		// Start secure protocol handlers
		syncService.RegisterProtocols(ctx)

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

			// Perform secure handshake and key exchange
			trusted, err := syncService.VerifyAndExchangeKeys(ctx, info.ID)
			if err != nil {
				return fmt.Errorf("key exchange failed: %v", err)
			}

			if !trusted {
				// If not automatically trusted, prompt user
				fmt.Printf("\n⚠️  New peer detected!\n")
				fmt.Printf("Peer ID: %s\n", info.ID.String())

				fingerprint, err := syncService.GetPeerFingerprint(info.ID)
				if err == nil {
					fmt.Printf("Fingerprint: %s\n", fingerprint)
				}

				if !promptForTrust() {
					return fmt.Errorf("sync canceled - peer not trusted")
				}

				// Add peer to trusted list
				err = syncService.AddTrustedPeer(ctx, info.ID)
				if err != nil {
					return fmt.Errorf("failed to add trusted peer: %v", err)
				}
			}

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
		defer func() { _ = discovery.Stop() }()

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

			// Perform secure handshake and key exchange
			trusted, err := syncService.VerifyAndExchangeKeys(ctx, peerInfo.ID)
			if err != nil {
				return fmt.Errorf("key exchange failed: %v", err)
			}

			if !trusted {
				// If not automatically trusted, prompt user
				fmt.Printf("\n⚠️  New peer detected!\n")
				fmt.Printf("Peer ID: %s\n", peerInfo.ID.String())

				fingerprint, err := syncService.GetPeerFingerprint(peerInfo.ID)
				if err == nil {
					fmt.Printf("Fingerprint: %s\n", fingerprint)
				}

				if !promptForTrust() {
					return fmt.Errorf("sync canceled - peer not trusted")
				}

				// Add peer to trusted list
				err = syncService.AddTrustedPeer(ctx, peerInfo.ID)
				if err != nil {
					return fmt.Errorf("failed to add trusted peer: %v", err)
				}
			}

			fmt.Printf("🔄 Starting sync with peer: %s\n", peerInfo.ID.String())

			// Sync with the peer
			result, err := syncService.SyncWithPeer(ctx, peerInfo.ID)
			if err != nil {
				// Save failed sync
				record := history.SyncRecord{
					ID:        uuid.New().String(),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					PeerID:    peerInfo.ID.String(),
					PeerName:  "vault-peer",
					Status:    "failed",
					Error:     err.Error(),
				}
				_ = history.AddRecord(filepath.Join(vaultRoot, ".sietch", "sync-history.json"), record)
				return fmt.Errorf("sync failed: %v", err)
			}

			// Display sync results
			displaySyncResults(result)
			
			// Create a sync history record
			record := history.SyncRecord{
				ID:                 uuid.New().String(),
				Timestamp:          time.Now().UTC().Format(time.RFC3339),
				PeerID:             peerInfo.ID.String(),
				PeerName:           "vault-peer", // optional: put a real name if you have it
				FilesTransferred:   result.FileCount,
				ChunksTransferred:  result.ChunksTransferred,
				ChunksDeduplicated: result.ChunksDeduplicated,
				BytesTransferred:   result.BytesTransferred,
				DurationMs:         result.Duration.Milliseconds(),
				Status:             "success",
				Files: FileInfosToPaths(result.FileListWithSizes), // make sure result has this info
				Error:              "",
			}

			// Save the record to .sietch/sync-history.json
			_ = history.AddRecord(filepath.Join(vaultRoot, ".sietch", "sync-history.json"), record)

		case <-timeoutCtx.Done():
			return fmt.Errorf("discovery timed out after %d seconds, no peers found", timeout)
		}

		return nil
	},
}

// loadRSAKeys loads the RSA key pair from the vault
func loadRSAKeys(vaultRoot string, cfg *config.VaultConfig) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Get path to private key
	privateKeyPath := filepath.Join(vaultRoot, cfg.Sync.RSA.PrivateKeyPath)

	// Read private key file
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Decode PEM block
	block, _ := pem.Decode(privateKeyData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	// Parse private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get public key from private key
	publicKey := &privateKey.PublicKey

	return privateKey, publicKey, nil
}

// rsaToLibp2pPrivateKey converts a Go RSA private key to libp2p format
func rsaToLibp2pPrivateKey(privateKey *rsa.PrivateKey) (crypto.PrivKey, error) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	return crypto.UnmarshalRsaPrivateKey(privateKeyBytes)
}

func FileInfosToPaths(files []p2p.FileInfo) []string {
    paths := make([]string, len(files))
    for i, f := range files {
        paths[i] = f.Path
    }
    return paths
}

// promptForTrust asks the user whether to trust a new peer
func promptForTrust() bool {
	fmt.Print("\nDo you want to trust this peer? (y/n): ")
	var response string
	_, _ = fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
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

var historyCmd = &cobra.Command{
    Use:   "history",
    Short: "Show recent syncs",
    RunE: func(cmd *cobra.Command, args []string) error {
        vaultRoot, err := fs.FindVaultRoot()
        if err != nil {
            return fmt.Errorf("not inside a vault: %v", err)
        }

        historyPath := filepath.Join(vaultRoot, ".sietch", "sync-history.json")
        h, _ := history.LoadHistory(historyPath)

        fmt.Println("Last syncs:")
        for i, r := range h.Syncs {
            if i >= 10 { break }
            fmt.Printf("%s  %s  %d files  %d B  %s\n", r.Timestamp, r.PeerName, r.FilesTransferred, r.BytesTransferred, r.Status)
        }
        return nil
    },
}


func init() {
	rootCmd.AddCommand(syncCmd)

	// Add command flags
	syncCmd.Flags().IntP("port", "p", 0, "Port to use for libp2p (0 for random port)")
	syncCmd.Flags().IntP("timeout", "t", 60, "Discovery timeout in seconds (for auto-discovery)")
	syncCmd.Flags().BoolP("force-trust", "f", false, "Automatically trust new peers without prompting")
	syncCmd.Flags().BoolP("read-only", "r", false, "Only receive files, don't send")

	// Register history subcommand
    syncCmd.AddCommand(historyCmd)
}
