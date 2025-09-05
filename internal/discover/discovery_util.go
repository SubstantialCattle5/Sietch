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
func CreateSyncService(h host.Host, vaultMgr *config.Manager, vaultConfig *config.VaultConfig, vaultPath string) (*p2p.SyncService, error) {
	if vaultConfig.Sync.Enabled && vaultConfig.Sync.RSA != nil {
		privateKey, publicKey, rsaConfig, err := keys.LoadRSAKeys(vaultPath, vaultConfig.Sync.RSA)
		if err != nil {
			return nil, fmt.Errorf("failed to load RSA keys: %v", err)
		}

		syncService, err := p2p.NewSecureSyncService(h, vaultMgr, privateKey, publicKey, rsaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync service: %v", err)
		}

		fmt.Println("🔐 RSA key exchange enabled with fingerprint:", rsaConfig.Fingerprint)
		return syncService, nil
	} else {
		syncService, err := p2p.NewSyncService(h, vaultMgr)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync service: %v", err)
		}

		fmt.Println("⚠️ Warning: RSA key exchange not enabled in vault config")
		return syncService, nil
	}
}

func SetupDiscovery(ctx context.Context, h host.Host) (*p2p.MDNSDiscovery, <-chan peer.AddrInfo, error) {
	factory := p2p.NewFactory()

	discovery, err := factory.CreateMDNS(h)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mDNS discovery service: %v", err)
	}

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
func RunDiscoveryLoop(ctx context.Context, h host.Host, syncService *p2p.SyncService,
	peerChan <-chan peer.AddrInfo, timeout int, continuous bool,
) error {
	var timeoutChan <-chan time.Time
	if !continuous {
		timeoutChan = time.After(time.Duration(timeout) * time.Second)
		fmt.Printf("   Discovery will run for %d seconds. Press Ctrl+C to stop earlier.\n\n", timeout)
	} else {
		fmt.Println("   Discovery will run until interrupted. Press Ctrl+C to stop.")
		fmt.Println()
	}

	discoveredPeers := make(map[string]bool)
	peerCount := 0

	for {
		select {
		case p, ok := <-peerChan:
			if !ok {
				return nil
			}

			if p.ID == h.ID() || discoveredPeers[p.ID.String()] {
				continue
			}

			discoveredPeers[p.ID.String()] = true
			peerCount++

			handleDiscoveredPeer(ctx, h, syncService, p, peerCount)

		case <-timeoutChan:
			fmt.Printf("\n⌛ Discovery timeout reached after %d seconds.\n", timeout)
			if peerCount == 0 {
				fmt.Println("   No Sietch vaults were discovered on the local network.")
			} else {
				fmt.Printf("   Discovered %d Sietch vault(s) on the local network.\n", peerCount)
			}
			return nil

		case <-ctx.Done():
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
func handleDiscoveredPeer(ctx context.Context, h host.Host, syncService *p2p.SyncService,
	p peer.AddrInfo, peerCount int,
) {
	fmt.Printf("✅ Discovered peer #%d\n", peerCount)
	fmt.Printf("   ID: %s\n", p.ID.String())
	fmt.Println("   Addresses:")
	for _, addr := range p.Addrs {
		fmt.Printf("     - %s\n", addr.String())
	}

	fmt.Printf("   Connecting and exchanging keys... ")

	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)
	defer connectCancel()

	if err := h.Connect(connectCtx, p); err != nil {
		fmt.Printf("connection failed: %v\n", err)
		return
	}

	trusted, err := syncService.VerifyAndExchangeKeys(connectCtx, p.ID)
	if err != nil {
		fmt.Printf("key exchange failed: %v\n", err)
		return
	}

	if trusted {
		fmt.Println("key exchange successful ✓")

		fingerprint, _ := syncService.GetPeerFingerprint(p.ID)
		fmt.Printf("   Peer fingerprint: %s\n", fingerprint)

		if err := syncService.AddTrustedPeer(ctx, p.ID); err != nil {
			fmt.Printf("   Failed to save trusted peer: %v\n", err)
		} else {
			fmt.Println("   Peer added to trusted peers list ✓")
		}
	} else {
		fmt.Println("peer not trusted")
	}
}
