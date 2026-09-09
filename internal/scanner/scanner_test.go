package scanner

import (
	"context"
	"net"
	"testing"
	"time"
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

func TestGenerateIPsSubnetLargerThanCount(t *testing.T) {
	// /24 contains 256 IPs; requesting count 128 should slice cleanly without INPUT_TOO_LARGE
	ips, _, err := GenerateIPs("192.168.1.0/24", 128)
	if err != nil {
		t.Fatalf("GenerateIPs failed for count smaller than subnet: %v", err)
	}
	if len(ips) != 128 {
		t.Fatalf("expected 128 IPs, got %d", len(ips))
	}
}

func TestGenerateIPsSingleHost(t *testing.T) {
	ips, _, err := GenerateIPs("192.168.1.1", 0)
	if err != nil {
		t.Fatalf("GenerateIPs failed for single host: %v", err)
	}
	if len(ips) != 1 {
		t.Fatalf("expected 1 IP for single host, got %d", len(ips))
	}
	if ips[0].String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ips[0].String())
	}
}

func TestSweepMultiplePings(t *testing.T) {
	ctx := context.Background()
	ips := []net.IP{net.ParseIP("127.0.0.1")}
	cfg := DefaultScannerConfig()
	cfg.Concurrency = 1
	cfg.Pings = 2

	results := Sweep(ctx, ips, cfg, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status == StatusOffline {
		t.Errorf("expected 127.0.0.1 to be active with 2 pings")
	}
}

func TestIsPrivateOrLocal(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"172.32.0.1", false},
		{"192.168.1.1", true},
		{"192.168.254.254", true},
		{"127.0.0.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := IsPrivateOrLocal(ip)
		if got != tt.want {
			t.Errorf("IsPrivateOrLocal(%s) = %v; want %v", tt.ip, got, tt.want)
		}
	}
}

func TestResolveDefaultTimeout(t *testing.T) {
	rfc1918 := []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("10.0.0.1")}
	if got := ResolveDefaultTimeout(rfc1918); got != 150*time.Millisecond {
		t.Errorf("Expected 150ms for RFC1918 subnets, got %v", got)
	}

	mixed := []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("8.8.8.8")}
	if got := ResolveDefaultTimeout(mixed); got != 400*time.Millisecond {
		t.Errorf("Expected 400ms for mixed/public targets, got %v", got)
	}

	publicOnly := []net.IP{net.ParseIP("1.1.1.1")}
	if got := ResolveDefaultTimeout(publicOnly); got != 400*time.Millisecond {
		t.Errorf("Expected 400ms for public targets, got %v", got)
	}
}

func TestSweepConcurrentMultiPings(t *testing.T) {
	ctx := context.Background()
	ips := []net.IP{net.ParseIP("127.0.0.1")}
	cfg := DefaultScannerConfig()
	cfg.Concurrency = 1
	cfg.Pings = 3
	cfg.Timeout = 150 * time.Millisecond

	start := time.Now()
	results := Sweep(ctx, ips, cfg, nil)
	elapsed := time.Since(start)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status == StatusOffline {
		t.Errorf("expected 127.0.0.1 to be active with 3 pings")
	}

	// 3 concurrent pings on loopback should take << 100ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected concurrent pings to complete rapidly on loopback, took %v", elapsed)
	}
}

func TestSweepGatewayHighlight(t *testing.T) {
	ctx := context.Background()
	ips := []net.IP{net.ParseIP("127.0.0.1")}

	// Without gateway set, a fast loopback response should be StatusOnline (not StatusHighlight)
	cfg := DefaultScannerConfig()
	cfg.Concurrency = 1
	cfg.Pings = 1
	results := Sweep(ctx, ips, cfg, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusOnline {
		t.Errorf("expected StatusOnline for fast host without GatewayIP, got %v", results[0].Status)
	}

	// With gateway set to 127.0.0.1, it should be StatusHighlight
	cfg.GatewayIP = net.ParseIP("127.0.0.1")
	results = Sweep(ctx, ips, cfg, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusHighlight {
		t.Errorf("expected StatusHighlight for GatewayIP, got %v", results[0].Status)
	}
}

func TestBroadcastDetection(t *testing.T) {
	// Test ExtractBroadcastIPs for /24
	b24 := ExtractBroadcastIPs("192.168.1.0/24")
	if len(b24) != 1 || b24[0].String() != "192.168.1.255" {
		t.Errorf("expected 192.168.1.255 for /24, got %v", b24)
	}

	// Test ExtractBroadcastIPs for /28
	b28 := ExtractBroadcastIPs("10.0.0.0/28")
	if len(b28) != 1 || b28[0].String() != "10.0.0.15" {
		t.Errorf("expected 10.0.0.15 for /28, got %v", b28)
	}

	// Test IsIPv4Broadcast
	if !IsIPv4Broadcast(net.ParseIP("255.255.255.255"), nil) {
		t.Error("expected 255.255.255.255 to be broadcast")
	}
	if !IsIPv4Broadcast(net.ParseIP("192.168.1.255"), nil) {
		t.Error("expected 192.168.1.255 to be broadcast")
	}
	if !IsIPv4Broadcast(net.ParseIP("10.0.0.15"), b28) {
		t.Error("expected 10.0.0.15 to be broadcast with b28 list")
	}
	if IsIPv4Broadcast(net.ParseIP("192.168.1.1"), nil) {
		t.Error("expected 192.168.1.1 NOT to be broadcast")
	}
}

func TestSweepSkipsBroadcast(t *testing.T) {
	ctx := context.Background()
	// Both normal loopback and broadcast addresses
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("192.168.1.255"),
		net.ParseIP("255.255.255.255"),
	}

	cfg := DefaultScannerConfig()
	cfg.Concurrency = 3
	cfg.Pings = 1
	cfg.Timeout = 50 * time.Millisecond

	results := Sweep(ctx, ips, cfg, nil)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// 127.0.0.1 should respond
	if results[0].Status == StatusOffline {
		t.Errorf("expected 127.0.0.1 to be online, got %v", results[0].Status)
	}
	// Broadcast IPs must NOT be pinged and remain Offline
	if results[1].Status != StatusOffline {
		t.Errorf("expected 192.168.1.255 to remain offline (not pinged), got %v", results[1].Status)
	}
	if results[2].Status != StatusOffline {
		t.Errorf("expected 255.255.255.255 to remain offline (not pinged), got %v", results[2].Status)
	}
}

func TestFormatDurationMS(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ms"},
		{500 * time.Microsecond, "0.5ms"},
		{250 * time.Microsecond, "0.25ms"},
		{50 * time.Microsecond, "0.05ms"},
		{5 * time.Millisecond, "5ms"},
		{12500 * time.Microsecond, "12.5ms"},
		{100 * time.Millisecond, "100ms"},
	}

	for _, tt := range tests {
		got := FormatDurationMS(tt.d)
		if got != tt.want {
			t.Errorf("FormatDurationMS(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
