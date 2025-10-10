package discover

import (
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/multiformats/go-multiaddr"
)

// AddressPriority defines the priority levels for different address types
type AddressPriority int

const (
	PriorityVirtual   AddressPriority = 0 // Filtered out (Docker, VPN, etc.)
	PriorityOther     AddressPriority = 1 // Shown only with --all-addresses
	PriorityIPv6      AddressPriority = 2 // Limited to 1 address
	PriorityLocalhost AddressPriority = 3 // Always shown
	PriorityLAN       AddressPriority = 4 // Primary LAN addresses
)

// FilteredAddress represents an address with its priority and display information
type FilteredAddress struct {
	Original    multiaddr.Multiaddr
	Priority    AddressPriority
	DisplayAddr string
	Label       string
}

// AddressFilter handles filtering and prioritizing network addresses
type AddressFilter struct {
	showAll bool
}

// NewAddressFilter creates a new address filter
func NewAddressFilter(showAll bool) *AddressFilter {
	return &AddressFilter{showAll: showAll}
}

// FilterAddresses filters and prioritizes a list of multiaddresses
func (af *AddressFilter) FilterAddresses(addrs []multiaddr.Multiaddr) []FilteredAddress {
	var filtered []FilteredAddress

	for _, addr := range addrs {
		if fa := af.categorizeAddress(addr); fa != nil {
			filtered = append(filtered, *fa)
		}
	}

	// Sort by priority (highest first)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority > filtered[j].Priority
		}
		// Secondary sort by address string for consistency
		return filtered[i].DisplayAddr < filtered[j].DisplayAddr
	})

	if af.showAll {
		return filtered
	}

	return af.applyFiltering(filtered)
}

// categorizeAddress determines the priority and display format for an address
func (af *AddressFilter) categorizeAddress(addr multiaddr.Multiaddr) *FilteredAddress {
	// Extract IP and port from multiaddr
	ip, port := af.extractIPAndPort(addr)
	if ip == "" {
		return nil
	}

	fa := &FilteredAddress{
		Original: addr,
	}

	// Check for localhost
	if af.isLocalhost(ip) {
		fa.Priority = PriorityLocalhost
		fa.DisplayAddr = "localhost:" + port
		fa.Label = ""
		return fa
	}

	// Check for virtual interfaces (Docker, VPN, etc.)
	if af.isVirtualInterface(ip) {
		fa.Priority = PriorityVirtual
		fa.DisplayAddr = ip + ":" + port
		fa.Label = af.getVirtualInterfaceLabel(ip)
		return fa
	}

	// Check for private LAN addresses
	if af.isPrivateLAN(ip) {
		fa.Priority = PriorityLAN
		fa.DisplayAddr = ip + ":" + port
		fa.Label = "(primary)"
		return fa
	}

	// Check for IPv6
	if af.isIPv6(ip) {
		fa.Priority = PriorityIPv6
		fa.DisplayAddr = "[" + ip + "]:" + port
		fa.Label = ""
		return fa
	}

	// Everything else
	fa.Priority = PriorityOther
	fa.DisplayAddr = ip + ":" + port
	fa.Label = ""
	return fa
}

// applyFiltering applies the filtering rules when showAll is false
func (af *AddressFilter) applyFiltering(addresses []FilteredAddress) []FilteredAddress {
	var result []FilteredAddress
	ipv6Count := 0

	for _, addr := range addresses {
		switch addr.Priority {
		case PriorityLAN, PriorityLocalhost:
			// Always include LAN and localhost
			result = append(result, addr)
		case PriorityIPv6:
			// Include only one IPv6 address
			if ipv6Count == 0 {
				result = append(result, addr)
				ipv6Count++
			}
		case PriorityVirtual, PriorityOther:
			// Skip virtual and other addresses in filtered mode
			continue
		}
	}

	// If we have no addresses, include at least one non-virtual address
	if len(result) == 0 {
		for _, addr := range addresses {
			if addr.Priority > PriorityVirtual {
				result = append(result, addr)
				break
			}
		}
	}

	return result
}

// extractIPAndPort extracts IP and port from a multiaddr
func (af *AddressFilter) extractIPAndPort(addr multiaddr.Multiaddr) (string, string) {
	var ip, port string
	
	multiaddr.ForEach(addr, func(c multiaddr.Component) bool {
		switch c.Protocol().Code {
		case multiaddr.P_IP4, multiaddr.P_IP6:
			ip = c.Value()
		case multiaddr.P_TCP, multiaddr.P_UDP:
			port = c.Value()
		}
		return true
	})
	
	return ip, port
}

// isLocalhost checks if an IP is localhost
func (af *AddressFilter) isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1"
}

// isPrivateLAN checks if an IP is a private LAN address
func (af *AddressFilter) isPrivateLAN(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check for private IPv4 ranges
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range private {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			// Exclude Docker ranges from LAN classification
			if af.isDockerRange(ip) {
				return false
			}
			return true
		}
	}

	return false
}

// isVirtualInterface checks if an IP belongs to a virtual interface
func (af *AddressFilter) isVirtualInterface(ip string) bool {
	return af.isDockerRange(ip) || af.isVPNRange(ip)
}

// isDockerRange checks if an IP is in a Docker range
func (af *AddressFilter) isDockerRange(ip string) bool {
	// Common Docker ranges
	dockerRanges := []string{
		"172.17.0.0/16", // Default Docker bridge
		"172.18.0.0/16", // Docker custom networks
		"172.19.0.0/16",
		"172.20.0.0/16",
		"172.21.0.0/16",
		"172.22.0.0/16",
		"172.23.0.0/16",
		"172.24.0.0/16",
		"172.25.0.0/16",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range dockerRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	// Also check for Docker-like patterns in 192.168.x.1 ranges
	if matched, _ := regexp.MatchString(`^192\.168\.(224|240|208|176|144|112|80|48|16)\.1$`, ip); matched {
		return true
	}

	return false
}

// isVPNRange checks if an IP is in a VPN range
func (af *AddressFilter) isVPNRange(ip string) bool {
	// Common VPN ranges - this is a basic implementation
	// You might want to expand this based on your specific VPN software
	vpnPatterns := []string{
		`^10\.8\.0\.`, // OpenVPN default
		`^10\.9\.0\.`, // WireGuard common
		`^192\.168\.122\.`, // libvirt/KVM default
	}

	for _, pattern := range vpnPatterns {
		if matched, _ := regexp.MatchString(pattern, ip); matched {
			return true
		}
	}

	return false
}

// isIPv6 checks if an IP is IPv6
func (af *AddressFilter) isIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}

// getVirtualInterfaceLabel returns a label for virtual interfaces
func (af *AddressFilter) getVirtualInterfaceLabel(ip string) string {
	if af.isDockerRange(ip) {
		return "(docker)"
	}
	if af.isVPNRange(ip) {
		return "(vpn)"
	}
	return "(virtual)"
}

// CountFilteredAddresses returns the count of addresses that would be filtered out
func (af *AddressFilter) CountFilteredAddresses(addrs []multiaddr.Multiaddr) int {
	if af.showAll {
		return 0
	}

	filtered := af.FilterAddresses(addrs)
	return len(addrs) - len(filtered)
}