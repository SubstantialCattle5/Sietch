package discover

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
	"github.com/substantialcattle5/sietch/internal/p2p"
)

// createSyncService creates a sync service with or without RSA support
func CreateSyncService(host host.Host, vaultMgr *config.Manager, vaultConfig *config.VaultConfig, vaultPath string) (*p2p.SyncService, error) {
	if vaultConfig.Sync.Enabled && vaultConfig.Sync.RSA != nil {
		// Load RSA keys using the keys package
		privateKey, publicKey, rsaConfig, err := keys.LoadRSAKeys(vaultPath, vaultConfig.Sync.RSA)
		if err != nil {
			return nil, fmt.Errorf("failed to load RSA keys: %v", err)
		}

		// Create secure sync service
		syncService, err := p2p.NewSecureSyncService(host, vaultMgr, privateKey, publicKey, rsaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync service: %v", err)
		}

		fmt.Println("🔐 RSA key exchange enabled with fingerprint:", rsaConfig.Fingerprint)
		return syncService, nil
	} else {
		// Create basic sync service without RSA
		syncService, err := p2p.NewSyncService(host, vaultMgr)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync service: %v", err)
		}
		fmt.Println("⚠️ Warning: RSA key exchange not enabled in vault config")
		return syncService, nil
	}
}

func SetupDiscovery(ctx context.Context, host host.Host) (*p2p.MDNSDiscovery, <-chan peer.AddrInfo, error) {
	// Create the discovery factory
	factory := p2p.NewFactory()

	// Create and start the mDNS discovery service
	discovery, err := factory.CreateMDNS(host)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mDNS discovery service: %v", err)
	}

	// Add type assertion here
	mdnsDiscovery, ok := discovery.(*p2p.MDNSDiscovery)
	if !ok {
		return nil, nil, fmt.Errorf("discovery is not of type *p2p.MDNSDiscovery")
	}

	if err := mdnsDiscovery.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start mDNS discovery: %v", err)
	}

	return mdnsDiscovery, mdnsDiscovery.DiscoveredPeers(), nil
}

// runDiscoveryLoop processes discovered peers until timeout or interrupted
func RunDiscoveryLoop(ctx context.Context, host host.Host, syncService *p2p.SyncService,
	peerChan <-chan peer.AddrInfo, timeout int, continuous bool) error {
	// Set up timeout
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

	// Process discovered peers
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

			// Handle the peer
			handleDiscoveredPeer(ctx, host, syncService, peer, peerCount)

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
}

// handleDiscoveredPeer processes a newly discovered peer
func handleDiscoveredPeer(ctx context.Context, host host.Host, syncService *p2p.SyncService,
	peer peer.AddrInfo, peerCount int) {
	fmt.Printf("✅ Discovered peer #%d\n", peerCount)
	fmt.Printf("   ID: %s\n", peer.ID.String())
	fmt.Println("   Addresses:")
	for _, addr := range peer.Addrs {
		fmt.Printf("     - %s\n", addr.String())
	}

	// Connect to the peer and exchange keys
	fmt.Printf("   Connecting and exchanging keys... ")
	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)
	defer connectCancel()

	if err := host.Connect(connectCtx, peer); err != nil {
		fmt.Printf("connection failed: %v\n", err)
		return
	}

	// Exchange keys
	trusted, err := syncService.VerifyAndExchangeKeys(connectCtx, peer.ID)
	if err != nil {
		fmt.Printf("key exchange failed: %v\n", err)
		return
	}

	if trusted {
		fmt.Println("key exchange successful ✓")

		// Get and display fingerprint
		fingerprint, _ := syncService.GetPeerFingerprint(peer.ID)
		fmt.Printf("   Peer fingerprint: %s\n", fingerprint)

		// Save peer to trusted list
		if err := syncService.AddTrustedPeer(ctx, peer.ID); err != nil {
			fmt.Printf("   Failed to save trusted peer: %v\n", err)
		} else {
			fmt.Println("   Peer added to trusted peers list ✓")
		}
	} else {
		fmt.Println("peer not trusted")
	}
}
