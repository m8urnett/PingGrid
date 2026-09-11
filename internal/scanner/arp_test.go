package scanner

import (
	"net"
	"testing"
)

func TestIsUnicastMAC(t *testing.T) {
	tests := []struct {
		mac  string
		want bool
	}{
		{"00:15:5d:00:62:1b", true},
		{"50:9a:4c:75:3e:2f", true},
		{"ff:ff:ff:ff:ff:ff", false}, // Broadcast
		{"00:00:00:00:00:00", false}, // All zeroes
		{"01:00:5e:00:00:02", false}, // IPv4 Multicast
		{"01:00:5e:7f:fa:fa", false}, // IPv4 Multicast
		{"33:33:00:00:00:01", false}, // IPv6 Multicast
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsUnicastMAC(tt.mac)
		if got != tt.want {
			t.Errorf("IsUnicastMAC(%q) = %v, want %v", tt.mac, got, tt.want)
		}
	}
}

func TestEnrichResultsWithARPRequiresNewEntryForSilentStatus(t *testing.T) {
	ip := net.ParseIP("192.0.2.10")
	entry := ARPEntry{IP: ip, MAC: "00:15:5d:00:62:1b", Vendor: "Microsoft"}

	stale := []HostResult{{IP: ip, Status: StatusOffline}}
	enrichResultsWithARP(stale, map[string]ARPEntry{ip.String(): entry}, map[string]ARPEntry{ip.String(): entry})
	if stale[0].Status != StatusOffline {
		t.Fatalf("stale pre-scan cache entry promoted host to %s", stale[0].Status)
	}

	fresh := []HostResult{{IP: ip, Status: StatusOffline}}
	enrichResultsWithARP(fresh, nil, map[string]ARPEntry{ip.String(): entry})
	if fresh[0].Status != StatusSilent {
		t.Fatalf("new post-scan cache entry status = %s, want Silent", fresh[0].Status)
	}
}

func TestGetARPTable(t *testing.T) {
	table, err := GetARPTable()
	if err != nil {
		t.Fatalf("GetARPTable() error = %v", err)
	}
	t.Logf("GetARPTable returned %d entries", len(table))
	for ip, entry := range table {
		if !IsUnicastMAC(entry.MAC) {
			t.Errorf("Found non-unicast MAC in ARP table: %s -> %s", ip, entry.MAC)
		}
	}
}
