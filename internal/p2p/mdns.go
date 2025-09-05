package p2p

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/substantialcattle5/sietch/internal/config"
)

// MDNSDiscovery implements the config.Discovery interface using mDNS
type MDNSDiscovery struct {
	host     host.Host
	service  mdns.Service
	peerChan chan peer.AddrInfo
	ctx      context.Context
	cancel   context.CancelFunc
	mutex    sync.Mutex
	started  bool
	closed   bool
}

// mdnsNotifee gets notified when new peers are discovered via mDNS
type MdnsNotifee struct {
	peerChan chan peer.AddrInfo
}

// HandlePeerFound implements the mdns.Notifee interface
func (n *MdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.peerChan <- pi
}

// GetPeerChan returns the peer channel for notifee configuration
func (m *MDNSDiscovery) GetPeerChan() chan peer.AddrInfo {
	return m.peerChan
}

// SetService sets the mDNS service
func (m *MDNSDiscovery) SetService(service mdns.Service) {
	m.service = service
}

// NewMDNSDiscovery creates a new mDNS discovery service
func NewMDNSDiscovery(h host.Host) (*MDNSDiscovery, error) {
	// Create a discovery context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize the discovery service
	m := &MDNSDiscovery{
		host:     h,
		peerChan: make(chan peer.AddrInfo, 32), // Buffer for discovered peers
		ctx:      ctx,
		cancel:   cancel,
	}

	// Create the notifee that will handle discovered peers
	notifee := &MdnsNotifee{
		peerChan: m.peerChan,
	}

	// Create the mDNS service
	service := mdns.NewMdnsService(h, config.ServiceTag, notifee)
	m.service = service

	return m, nil
}

// Start initiates the discovery process
func (m *MDNSDiscovery) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil // Already started
	}

	if m.closed {
		return nil // Already closed
	}

	// Start the mDNS service
	if err := m.service.Start(); err != nil {
		return err
	}

	m.started = true
	return nil
}

// Stop halts the discovery process
func (m *MDNSDiscovery) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.started || m.closed {
		return nil
	}

	// Close the mDNS service
	if err := m.service.Close(); err != nil {
		return err
	}

	// Cancel our context and mark as closed
	m.cancel()
	m.closed = true
	close(m.peerChan)

	return nil
}

// DiscoveredPeers returns channel of found peers
func (m *MDNSDiscovery) DiscoveredPeers() <-chan peer.AddrInfo {
	return m.peerChan
}
