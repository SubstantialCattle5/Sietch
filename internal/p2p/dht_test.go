package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	// "github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/p2p"
	// "github.com/substantialcattle5/sietch/testutil"
)

func TestDHTDiscovery(t *testing.T) {
	ctx := context.Background()

	// 1. Define static relays
	relayAddrs := []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}

	var staticRelays []peer.AddrInfo
	for _, addr := range relayAddrs {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			t.Logf("Invalid relay address: %s, error: %v", addr, err)
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			t.Logf("Invalid relay peer info: %s, error: %v", addr, err)
			continue
		}

		staticRelays = append(staticRelays, *pi)
	}

	// 2. Create libp2p host with AutoRelay and NAT support
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0",
		),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
		libp2p.NATPortMap(),
	)
	if err != nil {
		t.Fatalf("Failed to create libp2p host: %v", err)
	}
	defer h.Close()

	t.Logf("Libp2p host created. ID: %s", h.ID())

	for _, addr := range h.Addrs() {
		t.Logf("Listening on: %s", addr)
	}

	// 3. Relay / NAT diagnostic check
	time.AfterFunc(20*time.Second, func() {
		t.Log("Performing relay/NAT diagnostic check...")

		if len(h.Addrs()) == 0 {
			t.Log("No listening addresses found after 20 seconds. Possible NAT issue.")
		} else {
			t.Log("Listening addresses after 20 seconds:")
			for _, addr := range h.Addrs() {
				if _, err := addr.ValueForProtocol(multiaddr.P_CIRCUIT); err == nil {
					t.Logf(" - Relay address: %s", addr)
				} else {
					t.Logf(" - Direct address: %s", addr)
				}
			}
		}
	})

	// 4. Load vault config to get bootstrap nodes
	// vaultPath := testutil.TempDir(t, "dht-test")
	// mgr, err := config.NewManager(vaultPath)
	// if err != nil {
	// 	t.Fatalf("Failed to create vault manager: %v", err)
	// }
	// vaultConfig, err := mgr.GetConfig()
	// if err != nil {
	// 	t.Fatalf("Failed to load vault config: %v", err)
	// }

	// bootstrapNodes := vaultConfig.Discovery.DHT.BootstrapNodes
	// if len(bootstrapNodes) == 0 {
	// 	t.Fatal("No bootstrap nodes configured in vault config")
	// }
	bootstrapNodeStrs := []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
	t.Logf("Using %d bootstrap nodes from config", len(bootstrapNodeStrs))

	var bootstrapNodes []multiaddr.Multiaddr
	for _, addr := range bootstrapNodeStrs {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			t.Fatalf("Invalid bootstrap node address: %s, error: %v", addr, err)
		}
		bootstrapNodes = append(bootstrapNodes, maddr)
	}

	// 5. Create DHTDiscovery instance
	discovery, err := p2p.NewDHTDiscovery(ctx, h, bootstrapNodes)
	if err != nil {
		t.Fatalf("Failed to create DHT discovery: %v", err)
	}
	t.Log("DHT discovery instance created")

	// 6. Start discovery
	if err := discovery.Start(ctx); err != nil {
		t.Fatalf("Failed to start DHT discovery: %v", err)
	}
	t.Log("DHT discovery started")

	// 7. Print discovered peers asynchronously
	go func() {
		for pi := range discovery.DiscoveredPeers() {
			t.Logf("Discovered peer: %s", pi.ID)
		}
	}()

	// 8. Print connection events
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			t.Logf("Connected to: %s", c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			t.Logf("Disconnected from: %s", c.RemotePeer())
		},
	})

	// 9. Wait for discovery (adjust duration if needed)
	t.Log("⏳ Waiting 30 seconds for peer discovery...")
	time.Sleep(30 * time.Second)

	// 10. Stop discovery
	if err := discovery.Stop(); err != nil {
		t.Fatalf("Failed to stop DHTDiscovery: %v", err)
	}
	t.Log("✅ DHTDiscovery stopped")
}
