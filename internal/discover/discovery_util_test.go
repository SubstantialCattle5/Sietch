package discover

import (
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/p2p"
)

// TestCreateSyncService_NoRSA ensures CreateSyncService returns a basic SyncService
// when RSA sync is disabled in the vault configuration.
func TestCreateSyncService_NoRSA(t *testing.T) {
	// Create a dummy libp2p host using the package helper. Use port 0 for an ephemeral port.
	h, err := p2p.CreateLibp2pHost(0)
	if err != nil {
		t.Fatalf("failed to create libp2p host: %v", err)
	}
	defer h.Close()

	// Create an empty vault manager that points to a temp directory via config.NewManager
	// Use an in-memory manager by creating a Manager for the current directory; the function
	// under test only checks VaultConfig values and will not perform I/O for the non-RSA path.
	vm, err := config.NewManager(".")
	if err != nil {
		t.Fatalf("failed to create vault manager: %v", err)
	}

	// Prepare a vault config with RSA disabled
	vc := &config.VaultConfig{}
	vc.Sync.Enabled = false

	svc, err := CreateSyncService(h, vm, vc, ".", false)
	if err != nil {
		t.Fatalf("CreateSyncService returned error: %v", err)
	}
	if svc == nil {
		t.Fatalf("CreateSyncService returned nil service")
	}

	_ = h
}
