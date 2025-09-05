package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"

	"github.com/substantialcattle5/sietch/internal/config"
)

type Factory struct{}

// NewFactory creates a new discovery factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateMDNS creates an mDNS discovery service
func (f *Factory) CreateMDNS(h host.Host) (config.Discovery, error) {
	return NewMDNSDiscovery(h)
}

// CreateDHT creates a DHT-based discovery service
func (f *Factory) CreateDHT(ctx context.Context, h host.Host, bootstrapAddrs []multiaddr.Multiaddr) (config.Discovery, error) {
	// This would be implemented later
	// For now just return an error
	return nil, fmt.Errorf("DHT discovery not yet implemented")
}
