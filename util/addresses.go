package util

import (
	"fmt"
	"net"
	"strings"

	"github.com/multiformats/go-multiaddr"
)

// AddressFilter provides functionality to filter and prioritize network addresses
type AddressFilter struct {
	ShowAll bool
}

// FilteredAddresses contains filtered addresses and metadata
type FilteredAddresses struct {
	Primary     []multiaddr.Multiaddr
	Hidden      []multiaddr.Multiaddr
	Total       int
	HiddenCount int
}

// NewAddressFilter creates a new address filter
func NewAddressFilter(showAll bool) *AddressFilter {
	return &AddressFilter{
		ShowAll: showAll,
	}
}

// FilterAddresses filters multiaddresses to show only relevant ones
func (af *AddressFilter) FilterAddresses(addrs []multiaddr.Multiaddr) *FilteredAddresses {
	if af.ShowAll {
		return &FilteredAddresses{
			Primary:     addrs,
			Hidden:      []multiaddr.Multiaddr{},
			Total:       len(addrs),
			HiddenCount: 0,
		}
	}

	primary := []multiaddr.Multiaddr{}
	hidden := []multiaddr.Multiaddr{}

	// First pass: categorize addresses
	for _, addr := range addrs {
		if af.IsRelevantAddress(addr) {
			primary = append(primary, addr)
		} else {
			hidden = append(hidden, addr)
		}
	}

	// Sort primary addresses by priority
	primary = af.sortByPriority(primary)

	return &FilteredAddresses{
		Primary:     primary,
		Hidden:      hidden,
		Total:       len(addrs),
		HiddenCount: len(hidden),
	}
}

// IsRelevantAddress determines if an address should be shown by default
func (af *AddressFilter) IsRelevantAddress(addr multiaddr.Multiaddr) bool {
	addrStr := addr.String()

	// Extract IP address
	ipStr := af.ExtractIPFromMultiaddr(addrStr)
	if ipStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Always show localhost
	if ip.IsLoopback() {
		return true
	}

	// Show private LAN addresses (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
	if af.IsPrivateLAN(ip) && !af.IsDockerAddress(ip) {
		return true
	}

	// Show one IPv6 address (prefer link-local or unique local)
	if ip.To4() == nil { // IPv6
		// Prefer link-local (fe80::/10) or unique local (fc00::/7)
		if ip.IsLinkLocalUnicast() || af.isUniqueLocal(ip) {
			return true
		}
	}

	return false
}

// ExtractIPFromMultiaddr extracts IP address from multiaddr string
func (af *AddressFilter) ExtractIPFromMultiaddr(addrStr string) string {
	// Format: /ip4/127.0.0.1/tcp/39295 or /ip6/::1/tcp/36104
	parts := strings.Split(addrStr, "/")
	if len(parts) >= 3 && (parts[1] == "ip4" || parts[1] == "ip6") {
		return parts[2]
	}
	return ""
}

// IsPrivateLAN checks if IP is in private LAN ranges
func (af *AddressFilter) IsPrivateLAN(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false // Not IPv4
	}

	// 192.168.0.0/16
	if ipv4[0] == 192 && ipv4[1] == 168 {
		return true
	}

	// 10.0.0.0/8
	if ipv4[0] == 10 {
		return true
	}

	// 172.16.0.0/12 (172.16.0.0 to 172.31.255.255)
	if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
		return true
	}

	return false
}

// IsDockerAddress checks if IP is likely a Docker bridge address
func (af *AddressFilter) IsDockerAddress(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false // Not IPv4
	}

	// Common Docker bridge ranges
	// 172.17.0.0/16 (default docker0)
	if ipv4[0] == 172 && ipv4[1] == 17 {
		return true
	}

	// Other Docker networks: 172.18-30.x.x (avoiding 172.16.x.x which is legitimate private)
	if ipv4[0] == 172 && ipv4[1] >= 18 && ipv4[1] <= 30 {
		return true
	}

	// Docker Desktop ranges
	// 192.168.65.0/24, 192.168.224.0/20, etc.
	if ipv4[0] == 192 && ipv4[1] == 168 {
		// Common Docker Desktop subnets
		if ipv4[2] == 65 || ipv4[2] == 224 || ipv4[2] == 240 {
			return true
		}
	}

	return false
}

// isUniqueLocal checks if IPv6 address is unique local (fc00::/7)
func (af *AddressFilter) isUniqueLocal(ip net.IP) bool {
	if ip.To4() != nil {
		return false // Not IPv6
	}
	// Unique local addresses start with fc or fd
	return ip[0] == 0xfc || ip[0] == 0xfd
}

// sortByPriority sorts addresses by priority
func (af *AddressFilter) sortByPriority(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	if len(addrs) <= 1 {
		return addrs
	}

	// Priority sorting: primary LAN first, then localhost, then IPv6
	localhost := []multiaddr.Multiaddr{}
	lan := []multiaddr.Multiaddr{}
	ipv6 := []multiaddr.Multiaddr{}

	for _, addr := range addrs {
		ipStr := af.ExtractIPFromMultiaddr(addr.String())
		if ipStr == "" {
			continue
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		if ip.IsLoopback() {
			localhost = append(localhost, addr)
		} else if ip.To4() != nil {
			lan = append(lan, addr)
		} else {
			ipv6 = append(ipv6, addr)
		}
	}

	// Show primary LAN addresses first, then localhost, then one IPv6
	result := append(lan, localhost...)
	// Only add first IPv6 address to avoid clutter
	if len(ipv6) > 0 {
		result = append(result, ipv6[0])
	}

	return result
}

// FormatAddresses formats addresses for display
func (af *AddressFilter) FormatAddresses(filtered *FilteredAddresses, nodeID string) []string {
	lines := []string{}

	for _, addr := range filtered.Primary {
		addrStr := addr.String()
		ipStr := af.ExtractIPFromMultiaddr(addrStr)

		// Format as IP:port for cleaner display
		if port := af.ExtractPortFromMultiaddr(addrStr); port != "" {
			formatted := af.FormatAddressWithLabel(ipStr, port)
			lines = append(lines, fmt.Sprintf("  - %s", formatted))
		} else {
			// Fallback to original format
			lines = append(lines, fmt.Sprintf("  - %s", addrStr))
		}
	}

	// Add hidden count if any
	if filtered.HiddenCount > 0 {
		lines = append(lines, fmt.Sprintf("  [+%d more, use --all-addresses to show]", filtered.HiddenCount))
	}

	return lines
}

// ExtractPortFromMultiaddr extracts port from multiaddr string
func (af *AddressFilter) ExtractPortFromMultiaddr(addrStr string) string {
	// Format: /ip4/127.0.0.1/tcp/39295
	parts := strings.Split(addrStr, "/")
	if len(parts) >= 5 && parts[3] == "tcp" {
		return parts[4]
	}
	return ""
}

// getAddressLabel returns a descriptive label for an IP address
func (af *AddressFilter) getAddressLabel(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	if ip.IsLoopback() {
		return "localhost"
	}

	if ip.To4() != nil && af.IsPrivateLAN(ip) && !af.IsDockerAddress(ip) {
		return "primary"
	}

	if ip.To4() == nil && (ip.IsLinkLocalUnicast() || af.isUniqueLocal(ip)) {
		return "ipv6"
	}

	return ""
}

// FormatAddressWithLabel formats an address with a friendly display
func (af *AddressFilter) FormatAddressWithLabel(ipStr, port string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Sprintf("%s:%s", ipStr, port)
	}

	// Use "localhost" instead of 127.0.0.1 for better readability
	if ip.IsLoopback() {
		if ip.To4() != nil {
			return fmt.Sprintf("localhost:%s", port)
		} else {
			return fmt.Sprintf("localhost:%s (ipv6)", port)
		}
	}

	formatted := fmt.Sprintf("%s:%s", ipStr, port)
	label := af.getAddressLabel(ipStr)
	if label != "" && label != "localhost" {
		formatted += fmt.Sprintf(" (%s)", label)
	}

	return formatted
}
