package discover

import (
	"testing"

	"github.com/multiformats/go-multiaddr"
)

func TestAddressFilter(t *testing.T) {
	// Create test addresses
	testAddrs := []string{
		"/ip4/127.0.0.1/tcp/39295",        // localhost
		"/ip4/192.168.0.133/tcp/39295",    // LAN
		"/ip4/172.17.0.1/tcp/39295",       // Docker
		"/ip4/172.21.0.1/tcp/39295",       // Docker
		"/ip4/10.8.0.1/tcp/39295",         // VPN
		"/ip6/::1/tcp/36104",              // IPv6 localhost
		"/ip6/2001:db8::1/tcp/36104",      // IPv6
		"/ip4/8.8.8.8/tcp/39295",          // Public IP
	}

	var addrs []multiaddr.Multiaddr
	for _, addrStr := range testAddrs {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			t.Fatalf("Failed to create multiaddr %s: %v", addrStr, err)
		}
		addrs = append(addrs, addr)
	}

	t.Run("ShowAll=true", func(t *testing.T) {
		filter := NewAddressFilter(true)
		filtered := filter.FilterAddresses(addrs)

		// Should return all addresses
		if len(filtered) != len(addrs) {
			t.Errorf("Expected %d addresses, got %d", len(addrs), len(filtered))
		}
	})

	t.Run("ShowAll=false", func(t *testing.T) {
		filter := NewAddressFilter(false)
		filtered := filter.FilterAddresses(addrs)

		// Should filter out Docker and VPN addresses
		expectedCount := 4 // localhost, LAN, IPv6 localhost (limited to 1), public IP
		if len(filtered) > expectedCount {
			t.Errorf("Expected at most %d addresses, got %d", expectedCount, len(filtered))
		}

		// Check that localhost and LAN are included
		hasLocalhost := false
		hasLAN := false
		for _, addr := range filtered {
			if addr.DisplayAddr == "localhost:39295" {
				hasLocalhost = true
			}
			if addr.DisplayAddr == "192.168.0.133:39295" && addr.Label == "(primary)" {
				hasLAN = true
			}
		}

		if !hasLocalhost {
			t.Error("Expected localhost address to be included")
		}
		if !hasLAN {
			t.Error("Expected LAN address to be included with (primary) label")
		}
	})

	t.Run("CountFilteredAddresses", func(t *testing.T) {
		filter := NewAddressFilter(false)
		count := filter.CountFilteredAddresses(addrs)

		// Should count Docker and VPN addresses as filtered
		if count < 3 { // At least Docker and VPN addresses
			t.Errorf("Expected at least 3 filtered addresses, got %d", count)
		}
	})
}

func TestAddressPrioritization(t *testing.T) {
	filter := NewAddressFilter(false)

	// Test localhost detection
	localhostAddr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/39295")
	fa := filter.categorizeAddress(localhostAddr)
	if fa.Priority != PriorityLocalhost {
		t.Errorf("Expected localhost priority, got %d", fa.Priority)
	}
	if fa.DisplayAddr != "localhost:39295" {
		t.Errorf("Expected localhost:39295, got %s", fa.DisplayAddr)
	}

	// Test LAN detection
	lanAddr, _ := multiaddr.NewMultiaddr("/ip4/192.168.0.133/tcp/39295")
	fa = filter.categorizeAddress(lanAddr)
	if fa.Priority != PriorityLAN {
		t.Errorf("Expected LAN priority, got %d", fa.Priority)
	}
	if fa.Label != "(primary)" {
		t.Errorf("Expected (primary) label, got %s", fa.Label)
	}

	// Test Docker detection
	dockerAddr, _ := multiaddr.NewMultiaddr("/ip4/172.17.0.1/tcp/39295")
	fa = filter.categorizeAddress(dockerAddr)
	if fa.Priority != PriorityVirtual {
		t.Errorf("Expected virtual priority for Docker, got %d", fa.Priority)
	}
}