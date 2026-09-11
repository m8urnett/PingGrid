package scanner

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// InterfaceInfo contains configuration details for a network interface.
type InterfaceInfo struct {
	Index        int
	Name         string
	HardwareAddr string
	IP           net.IP
	CIDR         string
	Netmask      net.IPMask
	Gateway      net.IP
	DNSServers   []net.IP
	DHCPServer   net.IP
	DHCPEnabled  bool
	MTU          int
	IsUp         bool
	IsLoopback   bool
	IsPrimary    bool
	Health       LinkHealth
}

// LinkHealth captures hardware description, link speed/duplex, MTU, DHCP lease status,
// and Wi-Fi SSID/band/RSSI correlation for a network adapter.
type LinkHealth struct {
	AdapterModel  string        `json:"adapter_model,omitempty"`
	LinkSpeedBps  uint64        `json:"link_speed_bps,omitempty"`
	LinkSpeedStr  string        `json:"link_speed,omitempty"`
	Duplex        string        `json:"duplex,omitempty"`
	MTU           int           `json:"mtu,omitempty"`
	DHCPEnabled   bool          `json:"dhcp_enabled"`
	DHCPStatus    string        `json:"dhcp_status,omitempty"`
	DHCPLeaseLeft time.Duration `json:"-"`
	IsWireless    bool          `json:"is_wireless"`
	SSID          string        `json:"ssid,omitempty"`
	RadioType     string        `json:"radio_type,omitempty"`
	Band          string        `json:"band,omitempty"`
	Channel       int           `json:"channel,omitempty"`
	SignalPercent int           `json:"signal_percent,omitempty"`
	SignalDBm     int           `json:"signal_dbm,omitempty"`
}

// MarshalJSON exposes lease duration in milliseconds instead of time.Duration nanoseconds.
func (lh LinkHealth) MarshalJSON() ([]byte, error) {
	type linkHealthAlias LinkHealth
	return json.Marshal(struct {
		linkHealthAlias
		DHCPLeaseLeftMS int64 `json:"dhcp_lease_left_ms,omitempty"`
	}{
		linkHealthAlias: linkHealthAlias(lh),
		DHCPLeaseLeftMS: lh.DHCPLeaseLeft.Milliseconds(),
	})
}

// FormatSpeed formats link speed in bits per second to a human-readable string.
func FormatSpeed(bps uint64) string {
	if bps >= 1_000_000_000 {
		if bps%1_000_000_000 == 0 {
			return fmt.Sprintf("%d Gbps", bps/1_000_000_000)
		}
		return fmt.Sprintf("%.1f Gbps", float64(bps)/1_000_000_000)
	}
	if bps >= 1_000_000 {
		return fmt.Sprintf("%d Mbps", bps/1_000_000)
	}
	if bps >= 1_000 {
		return fmt.Sprintf("%d Kbps", bps/1_000)
	}
	return ""
}

// ChannelToBand resolves 802.11 Wi-Fi channel numbers to frequency band names.
func ChannelToBand(channel int) string {
	if channel >= 1 && channel <= 14 {
		return "2.4 GHz"
	}
	if channel >= 32 && channel <= 177 {
		return "5 GHz"
	}
	if channel > 180 {
		return "6 GHz"
	}
	return ""
}

// SignalPercentToDBm computes approximate RSSI dBm from Windows 802.11 signal quality percentage.
func SignalPercentToDBm(pct int) int {
	if pct <= 0 {
		return -100
	}
	if pct >= 100 {
		return -50
	}
	return (pct / 2) - 100
}

// FormatBanner generates a clean context banner for terminal output.
func (lh LinkHealth) FormatBanner(ii InterfaceInfo, plain bool) string {
	var b strings.Builder
	border := "─────────────────────────────────────────────────────────────────────────────"
	header := "── Link Health HUD ──────────────────────────────────────────────────────────"

	if !plain {
		header = "\033[96;1m──\033[0m \033[1mLink Health HUD\033[0m \033[96;1m" + strings.Repeat("─", 58) + "\033[0m"
		border = "\033[96;1m" + strings.Repeat("─", 77) + "\033[0m"
	}

	b.WriteString(header + "\n")

	// Line 1: Adapter Model & OS Name
	adapterName := ii.Name
	if lh.AdapterModel != "" && lh.AdapterModel != ii.Name {
		adapterName = fmt.Sprintf("%s (%s)", lh.AdapterModel, ii.Name)
	}
	if !plain {
		fmt.Fprintf(&b, "  \033[1mAdapter\033[0m : %s\n", adapterName)
	} else {
		fmt.Fprintf(&b, "  Adapter : %s\n", adapterName)
	}

	// Line 2: Wireless details if applicable
	if lh.IsWireless && lh.SSID != "" {
		var parts []string
		parts = append(parts, fmt.Sprintf("SSID: %s", lh.SSID))
		if lh.RadioType != "" {
			if lh.Band != "" && lh.Channel > 0 {
				parts = append(parts, fmt.Sprintf("%s (%s, Ch %d)", lh.RadioType, lh.Band, lh.Channel))
			} else if lh.Band != "" {
				parts = append(parts, fmt.Sprintf("%s (%s)", lh.RadioType, lh.Band))
			} else {
				parts = append(parts, lh.RadioType)
			}
		}
		if lh.SignalPercent > 0 {
			parts = append(parts, fmt.Sprintf("Signal: %d%% (%d dBm)", lh.SignalPercent, lh.SignalDBm))
		}
		if !plain {
			fmt.Fprintf(&b, "  \033[1mWireless\033[0m: \033[92m%s\033[0m\n", strings.Join(parts, " • "))
		} else {
			fmt.Fprintf(&b, "  Wireless: %s\n", strings.Join(parts, " • "))
		}
	}

	// Line 3: Link Speed, Duplex, MTU, DHCP
	var linkParts []string
	if lh.LinkSpeedStr != "" {
		speedText := lh.LinkSpeedStr
		if lh.Duplex != "" {
			speedText += " " + lh.Duplex
		}
		linkParts = append(linkParts, speedText)
	}
	if lh.MTU > 0 {
		linkParts = append(linkParts, fmt.Sprintf("MTU: %d", lh.MTU))
	}
	if lh.DHCPStatus != "" {
		linkParts = append(linkParts, fmt.Sprintf("DHCP: %s", lh.DHCPStatus))
	} else if ii.DHCPEnabled {
		dhcpStr := "Active"
		if ii.DHCPServer != nil {
			dhcpStr = fmt.Sprintf("Active (%s)", ii.DHCPServer)
		}
		linkParts = append(linkParts, fmt.Sprintf("DHCP: %s", dhcpStr))
	} else if ii.IP != nil {
		linkParts = append(linkParts, "DHCP: Disabled (Static)")
	}

	if len(linkParts) > 0 {
		if !plain {
			fmt.Fprintf(&b, "  \033[1mLink\033[0m    : %s\n", strings.Join(linkParts, " • "))
		} else {
			fmt.Fprintf(&b, "  Link    : %s\n", strings.Join(linkParts, " • "))
		}
	}

	// Line 4: Gateway, DNS, Primary IP
	var topoParts []string
	if ii.Gateway != nil {
		topoParts = append(topoParts, fmt.Sprintf("Gateway: %s", ii.Gateway))
	}
	if len(ii.DNSServers) > 0 {
		var dnsStrs []string
		for _, d := range ii.DNSServers {
			if d != nil {
				dnsStrs = append(dnsStrs, d.String())
			}
		}
		if len(dnsStrs) > 0 {
			topoParts = append(topoParts, fmt.Sprintf("DNS: %s", strings.Join(dnsStrs, ", ")))
		}
	}
	if ii.IP != nil {
		topoParts = append(topoParts, fmt.Sprintf("Primary IP: %s [Me]", ii.IP))
	}

	if len(topoParts) > 0 {
		if !plain {
			fmt.Fprintf(&b, "  \033[1mTopology\033[0m: %s\n", strings.Join(topoParts, " • "))
		} else {
			fmt.Fprintf(&b, "  Topology: %s\n", strings.Join(topoParts, " • "))
		}
	}

	b.WriteString(border + "\n")
	return b.String()
}

// SubnetPrefixLength returns the number of bits in the interface subnet mask.
func (ii InterfaceInfo) SubnetPrefixLength() int {
	if ii.Netmask == nil {
		return 0
	}
	ones, _ := ii.Netmask.Size()
	return ones
}

// SweepCIDR returns a safe CIDR block for sweeping. If the actual mask is broader
// than /22 (e.g. /16 or /8), it safely constrains to a /24 encompassing the IP
// to prevent accidentally pinging tens of thousands or millions of addresses in zero-config mode.
func (ii InterfaceInfo) SweepCIDR() string {
	if ii.IP == nil {
		return ii.CIDR
	}
	ones := ii.SubnetPrefixLength()
	if ones >= 22 && ones <= 30 {
		return ii.CIDR
	}
	// Derive standard 256-host /24 block encompassing local address
	ip4 := ii.IP.To4()
	if ip4 != nil {
		base := ip4.Mask(net.CIDRMask(24, 32))
		return fmt.Sprintf("%s/24", base.String())
	}
	return ii.CIDR
}

// GetPrimaryLocalIP discovers the primary egress local IPv4 address by probing routing to 8.8.8.8:53.
// No UDP packets are transmitted over the wire; only the kernel route table is consulted.
func GetPrimaryLocalIP() net.IP {
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4(8, 8, 8, 8),
		Port: 53,
	})
	if err != nil {
		return nil
	}
	defer func() {
		_ = conn.Close()
	}()
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || localAddr == nil {
		return nil
	}
	return localAddr.IP.To4()
}

// GetNetworkInterfaces queries all active and inactive local network interfaces
// and identifies IPv4 addresses, subnet masks, gateways, and the primary routing interface.
func GetNetworkInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	primaryIP := GetPrimaryLocalIP()
	primaryGW, primaryIfIndex := getPlatformDefaultGateway(nil)

	var list []InterfaceInfo

	for _, iface := range ifaces {
		info := InterfaceInfo{
			Index:        iface.Index,
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			IsUp:         iface.Flags&net.FlagUp != 0,
			IsLoopback:   iface.Flags&net.FlagLoopback != 0,
		}

		addrs, err := iface.Addrs()
		if err != nil {
			list = append(list, info)
			continue
		}

		// Find first valid IPv4 address for this interface
		var foundIPv4 bool
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}

			ones, bits := ipNet.Mask.Size()
			base := ip4.Mask(ipNet.Mask)

			info.IP = ip4
			info.Netmask = ipNet.Mask
			if bits == 32 && ones > 0 {
				info.CIDR = fmt.Sprintf("%s/%d", base.String(), ones)
			} else {
				info.CIDR = fmt.Sprintf("%s/24", ip4.Mask(net.CIDRMask(24, 32)).String())
			}

			// Check if this interface is the primary egress interface
			if primaryIP != nil && ip4.Equal(primaryIP) {
				info.IsPrimary = true
			} else if primaryIfIndex > 0 && iface.Index == primaryIfIndex {
				info.IsPrimary = true
			}

			// Determine gateway
			if info.IsPrimary && primaryGW != nil {
				info.Gateway = primaryGW
			}

			info.DNSServers = getPlatformDNSServers()
			dhcpIP, dhcpEnabled := getPlatformDHCPInfo(iface.Index)
			info.DHCPEnabled = dhcpEnabled
			info.DHCPServer = dhcpIP
			info.Health = getPlatformLinkHealth(iface.Index, iface.Name, info.MTU, info.DHCPEnabled, info.DHCPServer)
			if info.Health.MTU == 0 && info.MTU > 0 {
				info.Health.MTU = info.MTU
			}
			if info.Health.DHCPEnabled {
				info.DHCPEnabled = true
			}

			foundIPv4 = true
			break
		}

		if !foundIPv4 && info.IsLoopback {
			// Check for loopback IPv4
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				ip4 := ipNet.IP.To4()
				if ip4 != nil && ip4.IsLoopback() {
					info.IP = ip4
					info.Netmask = ipNet.Mask
					info.CIDR = "127.0.0.1/8"
					break
				}
			}
		}

		if info.Health.AdapterModel == "" {
			info.Health = getPlatformLinkHealth(iface.Index, iface.Name, info.MTU, info.DHCPEnabled, info.DHCPServer)
			if info.Health.MTU == 0 && info.MTU > 0 {
				info.Health.MTU = info.MTU
			}
		}

		list = append(list, info)
	}

	return list, nil
}

// FindInterface matches an interface by numeric index or exact name. A unique
// case-insensitive substring is accepted; ambiguous matches fail explicitly.
func FindInterface(nameOrIndex string) (*InterfaceInfo, error) {
	ifaces, err := GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	query := strings.TrimSpace(nameOrIndex)
	if query == "" {
		return nil, fmt.Errorf("interface name or index cannot be empty")
	}

	// Try parsing as integer index
	if idx, err := strconv.Atoi(query); err == nil {
		for i := range ifaces {
			if ifaces[i].Index == idx {
				return &ifaces[i], nil
			}
		}
	}

	// Exact name match (case-insensitive)
	for i := range ifaces {
		if strings.EqualFold(ifaces[i].Name, query) {
			return &ifaces[i], nil
		}
	}

	// Unique substring name match (case-insensitive)
	var matches []int
	for i := range ifaces {
		if strings.Contains(strings.ToLower(ifaces[i].Name), strings.ToLower(query)) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 1 {
		return &ifaces[matches[0]], nil
	}
	if len(matches) > 1 {
		var names []string
		for _, i := range matches {
			names = append(names, ifaces[i].Name)
		}
		return nil, fmt.Errorf("ambiguous interface %q matches: %s", nameOrIndex, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("network interface %q not found", nameOrIndex)
}

// DetectLocalTopology locates the primary active network interface and returns its full InterfaceInfo.
func DetectLocalTopology() (InterfaceInfo, error) {
	ifaces, err := GetNetworkInterfaces()
	if err != nil {
		return InterfaceInfo{}, err
	}

	// First pass: look for the primary interface with an active IPv4 address
	for _, iface := range ifaces {
		if iface.IsPrimary && iface.IsUp && iface.IP != nil && !iface.IsLoopback {
			return iface, nil
		}
	}

	// Second pass: any active non-loopback IPv4 interface
	for _, iface := range ifaces {
		if iface.IsUp && !iface.IsLoopback && iface.IP != nil && !iface.IP.IsLinkLocalUnicast() {
			return iface, nil
		}
	}

	return InterfaceInfo{}, fmt.Errorf("no active IPv4 network interface found")
}

// GetActiveSweepableInterfaces returns all active, non-loopback network interfaces
// that have an assigned non-link-local IPv4 address and sweepable CIDR block.
// The primary routing interface is always placed first.
func GetActiveSweepableInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	var active []InterfaceInfo
	var primary *InterfaceInfo

	for i := range ifaces {
		if ifaces[i].IsUp && !ifaces[i].IsLoopback && ifaces[i].IP != nil && !ifaces[i].IP.IsLinkLocalUnicast() && ifaces[i].SweepCIDR() != "" {
			if ifaces[i].IsPrimary && primary == nil {
				p := ifaces[i]
				primary = &p
			} else {
				active = append(active, ifaces[i])
			}
		}
	}

	if primary != nil {
		active = append([]InterfaceInfo{*primary}, active...)
	}

	if len(active) == 0 {
		return nil, fmt.Errorf("no active sweepable IPv4 network interfaces found")
	}

	return active, nil
}
