package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

// CreateLibp2pHost creates a new libp2p host listening on the specified port.
// If port is 0, the system will choose an available port.
func CreateLibp2pHost(port int) (host.Host, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d (must be 0-65535)", port)
	}

	opts := []libp2p.Option{
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
	}

	// Configure listening addresses
	listenAddrs := []string{}
	if port > 0 {
		listenAddrs = append(listenAddrs,
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip6/::/tcp/%d", port))
	} else {
		listenAddrs = append(listenAddrs,
			"/ip4/0.0.0.0/tcp/0",
			"/ip6/::/tcp/0")
	}

	opts = append(opts, libp2p.ListenAddrStrings(listenAddrs...))

	return libp2p.New(opts...)
}
