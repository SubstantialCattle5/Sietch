package discover

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DisplayHostAddresses prints the addresses the host is listening on with filtering
func DisplayHostAddresses(h host.Host, showAll bool) {
	filter := NewAddressFilter(showAll)
	filtered := filter.FilterAddresses(h.Addrs())
	
	fmt.Println("Listening on:")
	
	if len(filtered) == 0 {
		fmt.Println("  No suitable addresses found")
		return
	}

	for _, addr := range filtered {
		if showAll {
			// Show full multiaddr format with peer ID
			fmt.Printf("  %s/p2p/%s", addr.Original, h.ID().String())
			if addr.Label != "" {
				fmt.Printf(" %s", addr.Label)
			}
			fmt.Println()
		} else {
			// Show simplified format
			fmt.Printf("  %s", addr.DisplayAddr)
			if addr.Label != "" {
				fmt.Printf(" %s", addr.Label)
			}
			fmt.Println()
		}
	}

	// Show summary of filtered addresses if any were hidden
	if !showAll {
		hiddenCount := filter.CountFilteredAddresses(h.Addrs())
		if hiddenCount > 0 {
			fmt.Printf("  [+%d more, use --all-addresses to show]\n", hiddenCount)
		}
	}
}

// DisplayPeerAddresses prints the addresses for a discovered peer with filtering
func DisplayPeerAddresses(peerInfo peer.AddrInfo, showAll bool) {
	filter := NewAddressFilter(showAll)
	filtered := filter.FilterAddresses(peerInfo.Addrs)
	
	fmt.Println("   Addresses:")
	
	if len(filtered) == 0 {
		fmt.Println("     No suitable addresses found")
		return
	}

	for _, addr := range filtered {
		if showAll {
			// Show full multiaddr format
			fmt.Printf("     - %s", addr.Original)
			if addr.Label != "" {
				fmt.Printf(" %s", addr.Label)
			}
			fmt.Println()
		} else {
			// Show simplified format
			fmt.Printf("     - %s", addr.DisplayAddr)
			if addr.Label != "" {
				fmt.Printf(" %s", addr.Label)
			}
			fmt.Println()
		}
	}

	// Show summary of filtered addresses if any were hidden
	if !showAll {
		hiddenCount := filter.CountFilteredAddresses(peerInfo.Addrs)
		if hiddenCount > 0 {
			fmt.Printf("     [+%d more, use --all-addresses to show]\n", hiddenCount)
		}
	}
}