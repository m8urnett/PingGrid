package scanner

import (
	"context"
	"net"
	"testing"
)

func TestGenerateIPsFromCIDR(t *testing.T) {
	ips, _, err := GenerateIPs("192.168.1.0/28", 16)
	if err != nil {
		t.Fatalf("GenerateIPs failed: %v", err)
	}
	if len(ips) != 16 {
		t.Fatalf("expected 16 IPs, got %d", len(ips))
	}
	if ips[0].String() != "192.168.1.0" {
		t.Errorf("expected first IP 192.168.1.0, got %s", ips[0].String())
	}
	if ips[15].String() != "192.168.1.15" {
		t.Errorf("expected last IP 192.168.1.15, got %s", ips[15].String())
	}
}

func TestGenerateIPsFromBase(t *testing.T) {
	ips, _, err := GenerateIPs("10.0.0.1", 5)
	if err != nil {
		t.Fatalf("GenerateIPs failed: %v", err)
	}
	if len(ips) != 5 {
		t.Fatalf("expected 5 IPs, got %d", len(ips))
	}
	expected := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"}
	for i, want := range expected {
		if ips[i].String() != want {
			t.Errorf("at index %d expected %s, got %s", i, want, ips[i].String())
		}
	}
}

func TestSweepCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ips := []net.IP{net.ParseIP("127.0.0.1")}
	cfg := DefaultScannerConfig()
	cfg.Concurrency = 1

	results := Sweep(ctx, ips, cfg, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}
