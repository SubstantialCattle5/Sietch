package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// DHTDiscovery implements the config.Discovery interface using Kademlia DHT
type DHTDiscovery struct {
	host           host.Host
	dht            *dht.IpfsDHT // Kademlia DHT instance
	peerChan       chan peer.AddrInfo
	ctx            context.Context
	cancel         context.CancelFunc
	mutex          sync.Mutex
	started        bool
	closed         bool
	bootstrapPeers []peer.AddrInfo
}

// NewDHTDiscovery creates a new DHT based discovery service
func NewDHTDiscovery(ctx context.Context, h host.Host, bootstrapAddrs []multiaddr.Multiaddr) (*DHTDiscovery, error) {
	// Create a discovery context with cancellation
	ctx, cancel := context.WithCancel(ctx)

	// Parse bootstrap multiaddrs
	peers := ParseBootstrapAddrs(bootstrapAddrs)

	// Initialize the DHT discovery service
	d := &DHTDiscovery{
		host:           h,
		peerChan:       make(chan peer.AddrInfo, 32), // Buffer for discovered peers
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: peers,
	}

	return d, nil
}

func ParseBootstrapAddrs(bootstrapAddrs []multiaddr.Multiaddr) []peer.AddrInfo {
	peers := make([]peer.AddrInfo, 0, len(bootstrapAddrs))
	for _, maddr := range bootstrapAddrs {
		pi, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Println("Skipping invalid peer info:", maddr, err)
			continue
		}

		peers = append(peers, *pi)
	}

	if len(peers) == 0 {
		fmt.Println("No valid bootstrap peers provided, using default bootstrap peers")
		peers = dht.GetDefaultBootstrapPeerAddrInfos()
	}
	return peers
}

// Start initiates the DHT discovery process
func (d *DHTDiscovery) Start(ctx context.Context) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.started {
		return nil // Already started
	}

	if d.closed {
		return nil // Already closed
	}

	d.started = true

	// Bootstrap and Discovery logic
	if err := d.BootstrapDHT(); err != nil {
		return err
	}

	// Start peer discovery routine
	go d.DiscoverPeers()

	return nil
}

func (d *DHTDiscovery) BootstrapDHT() error {
	// Initialize Kademlia DHT
	kadDHT, err := dht.New(d.ctx, d.host)
	if err != nil {
		return err
	}
	d.dht = kadDHT

	for _, p := range d.bootstrapPeers {
		fmt.Println("Connecting to bootstrap peer:", p.ID)
		if err := d.host.Connect(d.ctx, p); err != nil {
			fmt.Println("Failed to connect:", err)
		} else {
			fmt.Println("Connected to bootstrap peer:", p.ID)
		}
	}

	// Start background DHT bootstrapping (refresh peers etc)
	if err := d.dht.Bootstrap(d.ctx); err != nil {
		fmt.Println("Failed to bootstrap DHT:", err)
	}
	return nil
}

// Stop halts the DHT discovery process
func (d *DHTDiscovery) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.started || d.closed {
		return nil
	}

	// Close the DHT service
	if d.dht != nil {
		if err := d.dht.Close(); err != nil {
			return err
		}
	}

	// Cancel our context and mark as closed
	d.cancel()
	d.closed = true
	close(d.peerChan)

	return nil
}

// DiscoveredPeers returns channel of found peers
func (d *DHTDiscovery) DiscoveredPeers() <-chan peer.AddrInfo {
	return d.peerChan
}

// FindPeers queries the DHT for peers and sends them to the peerChan
func (d *DHTDiscovery) FindPeers() {
	peers := d.dht.RoutingTable().ListPeers()

	fmt.Printf("DHT has %d peers in routing table\n", len(peers))

	for _, pid := range peers {
		if pid == d.host.ID() {
			continue // Skip self
		}

		addrs := d.host.Peerstore().Addrs(pid)
		if len(addrs) == 0 {
			continue
		}

		// Send peer info (non-blocking)
		select {
		case d.peerChan <- peer.AddrInfo{ID: pid, Addrs: addrs}:
			fmt.Println("✨ Discovered peer:", pid)
		default:
			// Skip if channel full
		}
	}
}

// DiscoverPeers periodically searches for peers in the DHT
func (d *DHTDiscovery) DiscoverPeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.FindPeers()
		}
	}
}
