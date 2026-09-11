package scanner

import (
	"net"
	"strings"
	"testing"
)

func TestGetNetworkInterfaces(t *testing.T) {
	ifaces, err := GetNetworkInterfaces()
	if err != nil {
		t.Fatalf("GetNetworkInterfaces failed: %v", err)
	}
	if len(ifaces) == 0 {
		t.Fatal("Expected at least one network interface")
	}

	var foundUp bool
	for _, iface := range ifaces {
		if iface.IsUp {
			foundUp = true
			break
		}
	}
	if !foundUp {
		t.Log("Warning: no 'Up' network interfaces found on test system")
	}
}

func TestSweepCIDRSafety(t *testing.T) {
	tests := []struct {
		name     string
		info     InterfaceInfo
		expected string
	}{
		{
			name: "Standard /24 LAN",
			info: InterfaceInfo{
				IP:      net.ParseIP("192.168.1.50"),
				CIDR:    "192.168.1.0/24",
				Netmask: net.CIDRMask(24, 32),
			},
			expected: "192.168.1.0/24",
		},
		{
			name: "Corporate /23 Subnet",
			info: InterfaceInfo{
				IP:      net.ParseIP("10.0.1.15"),
				CIDR:    "10.0.0.0/23",
				Netmask: net.CIDRMask(23, 32),
			},
			expected: "10.0.0.0/23",
		},
		{
			name: "Small /28 Subnet",
			info: InterfaceInfo{
				IP:      net.ParseIP("192.168.10.20"),
				CIDR:    "192.168.10.16/28",
				Netmask: net.CIDRMask(28, 32),
			},
			expected: "192.168.10.16/28",
		},
		{
			name: "Oversized /16 Subnet Safely Clamped to /24",
			info: InterfaceInfo{
				IP:      net.ParseIP("172.16.5.42"),
				CIDR:    "172.16.0.0/16",
				Netmask: net.CIDRMask(16, 32),
			},
			expected: "172.16.5.0/24",
		},
		{
			name: "Oversized /8 Subnet Safely Clamped to /24",
			info: InterfaceInfo{
				IP:      net.ParseIP("10.20.30.40"),
				CIDR:    "10.0.0.0/8",
				Netmask: net.CIDRMask(8, 32),
			},
			expected: "10.20.30.0/24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.info.SweepCIDR()
			if got != tc.expected {
				t.Errorf("SweepCIDR() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestDetectLocalSubnet(t *testing.T) {
	cidr, gw, err := DetectLocalSubnet()
	if err != nil {
		t.Skipf("Skipping DetectLocalSubnet test (no active network interface): %v", err)
	}
	if cidr == "" {
		t.Error("Expected non-empty CIDR from DetectLocalSubnet")
	}
	_, _, err = net.ParseCIDR(cidr)
	if err != nil {
		t.Errorf("DetectLocalSubnet returned invalid CIDR %q: %v", cidr, err)
	}
	if gw == nil {
		t.Error("Expected non-nil gateway from DetectLocalSubnet")
	}
}

func TestLinkHealthFormatting(t *testing.T) {
	// Speed formatting
	if s := FormatSpeed(1_000_000_000); s != "1 Gbps" {
		t.Errorf("FormatSpeed(1Gbps) = %q, want 1 Gbps", s)
	}
	if s := FormatSpeed(2_500_000_000); s != "2.5 Gbps" {
		t.Errorf("FormatSpeed(2.5Gbps) = %q, want 2.5 Gbps", s)
	}
	if s := FormatSpeed(100_000_000); s != "100 Mbps" {
		t.Errorf("FormatSpeed(100Mbps) = %q, want 100 Mbps", s)
	}

	// Wi-Fi Band
	if b := ChannelToBand(6); b != "2.4 GHz" {
		t.Errorf("ChannelToBand(6) = %q, want 2.4 GHz", b)
	}
	if b := ChannelToBand(36); b != "5 GHz" {
		t.Errorf("ChannelToBand(36) = %q, want 5 GHz", b)
	}
	if b := ChannelToBand(185); b != "6 GHz" {
		t.Errorf("ChannelToBand(185) = %q, want 6 GHz", b)
	}

	// RSSI dBm calculation
	if dbm := SignalPercentToDBm(100); dbm != -50 {
		t.Errorf("SignalPercentToDBm(100) = %d, want -50", dbm)
	}
	if dbm := SignalPercentToDBm(90); dbm != -55 {
		t.Errorf("SignalPercentToDBm(90) = %d, want -55", dbm)
	}
	if dbm := SignalPercentToDBm(50); dbm != -75 {
		t.Errorf("SignalPercentToDBm(50) = %d, want -75", dbm)
	}
}

func TestLinkHealthBanner(t *testing.T) {
	info := InterfaceInfo{
		Name:    "Ethernet 2",
		IP:      net.ParseIP("10.8.0.150"),
		Gateway: net.ParseIP("10.8.0.1"),
		DNSServers: []net.IP{
			net.ParseIP("10.8.0.1"),
		},
		Health: LinkHealth{
			AdapterModel: "Microsoft Hyper-V Network Adapter #2",
			LinkSpeedStr: "1 Gbps",
			Duplex:       "Full Duplex",
			MTU:          1500,
			DHCPEnabled:  true,
			DHCPStatus:   "Active (10.8.0.1)",
		},
	}

	banner := info.Health.FormatBanner(info, true)
	if !strings.Contains(banner, "Link Health HUD") {
		t.Errorf("Expected banner header, got:\n%s", banner)
	}
	if !strings.Contains(banner, "Microsoft Hyper-V Network Adapter #2") {
		t.Errorf("Expected adapter model in banner, got:\n%s", banner)
	}
	if !strings.Contains(banner, "1 Gbps Full Duplex") {
		t.Errorf("Expected link speed and duplex, got:\n%s", banner)
	}
	if !strings.Contains(banner, "Primary IP: 10.8.0.150 [Me]") {
		t.Errorf("Expected primary IP with role badge, got:\n%s", banner)
	}
}

func TestGetActiveSweepableInterfaces(t *testing.T) {
	active, err := GetActiveSweepableInterfaces()
	if err != nil {
		t.Skipf("Skipping TestGetActiveSweepableInterfaces (no active interface on test host): %v", err)
	}

	if len(active) == 0 {
		t.Fatal("Expected at least one active sweepable interface")
	}

	for _, iface := range active {
		if !iface.IsUp {
			t.Errorf("Interface %s is not Up", iface.Name)
		}
		if iface.IsLoopback {
			t.Errorf("Interface %s is loopback, expected non-loopback", iface.Name)
		}
		if iface.IP == nil {
			t.Errorf("Interface %s has nil IP", iface.Name)
		}
		if iface.SweepCIDR() == "" {
			t.Errorf("Interface %s has empty SweepCIDR", iface.Name)
		}
	}
}
