package config

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// ServiceTag is used in mDNS advertisements to discover peers
const ServiceTag = "sietch-vault-sync"

type Discovery interface {
	// Start initiates the discovery process
	Start(context.Context) error

	// Stop halts the discovery process
	Stop() error

	// DiscoveredPeers returns channel of found peers
	DiscoveredPeers() <-chan peer.AddrInfo
}

type DiscoveryFactory interface {
	CreateMDNS(host.Host) (Discovery, error)
	CreateDHT(context.Context, host.Host, []multiaddr.Multiaddr) (Discovery, error)
}
