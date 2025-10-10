package discover

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/p2p"
)

// TestHandleDiscoveredPeerExercisesPaths ensures handleDiscoveredPeer runs through
// the code paths where AddTrustedPeer fails (peer not present) and where the peer
// is already present in the temporary trusted list.
func TestHandleDiscoveredPeerExercisesPaths(t *testing.T) {
	// Create host
	h, err := p2p.CreateLibp2pHost(0)
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	// Create a vault manager for CreateSyncService; we can reuse the local dir
	vm, err := config.NewManager(".")
	if err != nil {
		t.Fatalf("failed to create vault manager: %v", err)
	}

	// Create non-RSA sync service
	vc := &config.VaultConfig{}
	svc, err := CreateSyncService(h, vm, vc, ".", false)
	if err != nil {
		t.Fatalf("CreateSyncService failed: %v", err)
	}

	// Prepare peer info that points to our own host (connect to self)
	p := peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()}

	// Case 1: peer not present in trustedPeers -> AddTrustedPeer will fail
	handleDiscoveredPeer(context.Background(), h, svc, p, 1, false)

	// We cannot access unexported fields of SyncService from here; ensure
	// the function returns without panic when called a second time.
	handleDiscoveredPeer(context.Background(), h, svc, p, 2, false)
}
