package scanner

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestReadARPCache(t *testing.T) {
	table, err := ReadARPCache()
	if err != nil {
		t.Fatalf("ReadARPCache failed: %v", err)
	}
	t.Logf("Read %d ARP cache entries", len(table))
	for ip, mac := range table {
		if ip == "" || mac == "" {
			t.Errorf("invalid ARP entry: ip=%q, mac=%q", ip, mac)
		}
	}
}

func TestParseProcNetARP(t *testing.T) {
	sample := []byte(`IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
192.168.1.50     0x1         0x0         00:00:00:00:00:00     *        eth0
10.8.0.1         0x1         0x2         11:22:33:44:55:66     *        eth0
`)
	table := parseProcNetARP(sample)
	if len(table) != 2 {
		t.Errorf("expected 2 complete entries, got %d", len(table))
	}
	if table["192.168.1.1"] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("unexpected MAC for 192.168.1.1: %s", table["192.168.1.1"])
	}
	if _, exists := table["192.168.1.50"]; exists {
		t.Errorf("did not expect incomplete entry 192.168.1.50 in table")
	}
}

func TestParseArpOutput(t *testing.T) {
	sample := `? (192.168.1.1) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
? (10.8.0.1) at 11-22-33-44-55-66 on en0 ifscope [ethernet]
? (192.168.1.200) at (incomplete) on en0 ifscope [ethernet]
`
	table := parseArpOutput(sample)
	if len(table) != 2 {
		t.Errorf("expected 2 complete entries, got %d", len(table))
	}
	if table["192.168.1.1"] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("unexpected MAC for 192.168.1.1: %s", table["192.168.1.1"])
	}
	if table["10.8.0.1"] != "11:22:33:44:55:66" {
		t.Errorf("unexpected MAC for 10.8.0.1: %s", table["10.8.0.1"])
	}
	if _, exists := table["192.168.1.200"]; exists {
		t.Errorf("did not expect incomplete entry in table")
	}
}

func TestTwoPhaseSweep(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("127.0.0.2"),
	}

	phase1Called := false
	var p1Duration time.Duration

	cfg := Config{
		Count:         len(ips),
		Concurrency:   2,
		Timeout:       50 * time.Millisecond,
		SlowThreshold: 100 * time.Millisecond,
		Pings:         1,
		UseARPCache:   true,
		OnPhase1Complete: func(results []HostResult, d time.Duration) {
			phase1Called = true
			p1Duration = d
			if len(results) != len(ips) {
				t.Errorf("expected %d results in Phase 1 snapshot, got %d", len(ips), len(results))
			}
		},
	}

	results := Sweep(context.Background(), ips, cfg, nil)
	if !phase1Called {
		t.Errorf("expected OnPhase1Complete to be called")
	}
	if p1Duration < 0 {
		t.Errorf("invalid phase 1 duration: %v", p1Duration)
	}
	if len(results) != len(ips) {
		t.Errorf("expected %d final results, got %d", len(ips), len(results))
	}
}
