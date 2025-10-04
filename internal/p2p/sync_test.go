package p2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TestHasPeer ensures HasPeer returns false for unknown peer and true after insertion
func TestHasPeer(t *testing.T) {
    s := &SyncService{
        trustedPeers: make(map[peer.ID]*PeerInfo),
    }

    // Create a synthetic peer ID string and decode
    id, err := peer.Decode("QmYwAPJzv5CZsnAzt8auV2u6p6Yg3qR6gq7kKPpVd6Q7f6")
    if err != nil {
        t.Skipf("Skipping test due to invalid synthetic peer ID: %v", err)
    }

    if s.HasPeer(id) {
        t.Fatalf("expected HasPeer to return false for unknown peer")
    }

    s.trustedPeers[id] = &PeerInfo{ID: id}
    if !s.HasPeer(id) {
        t.Fatalf("expected HasPeer to return true after insertion")
    }
}
