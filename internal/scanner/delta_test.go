package scanner

import (
	"net"
	"testing"
	"time"
)

func TestComputeDeltas(t *testing.T) {
	ip1 := net.IPv4(192, 168, 1, 1)
	ip2 := net.IPv4(192, 168, 1, 2)
	ip3 := net.IPv4(192, 168, 1, 3)

	prev := []HostResult{
		{IP: ip1, Status: StatusOffline},
		{IP: ip2, Status: StatusOnline, RTT: 10 * time.Millisecond},
		{IP: ip3, Status: StatusOnline, RTT: 15 * time.Millisecond},
	}

	curr := []HostResult{
		{IP: ip1, Status: StatusOnline, RTT: 5 * time.Millisecond}, // Joined
		{IP: ip2, Status: StatusOffline},                           // Dropped
		{IP: ip3, Status: StatusSlow, RTT: 120 * time.Millisecond}, // Changed to Slow
	}

	deltas := ComputeDeltas(prev, curr)
	if len(deltas) != 3 {
		t.Fatalf("expected 3 deltas, got %d", len(deltas))
	}

	if deltas[0].Kind != DeltaJoined || !deltas[0].IP.Equal(ip1) {
		t.Errorf("delta 0 expected joined for ip1, got %+v", deltas[0])
	}
	if deltas[1].Kind != DeltaDropped || !deltas[1].IP.Equal(ip2) {
		t.Errorf("delta 1 expected dropped for ip2, got %+v", deltas[1])
	}
	if deltas[2].Kind != DeltaChanged || !deltas[2].IP.Equal(ip3) {
		t.Errorf("delta 2 expected changed for ip3, got %+v", deltas[2])
	}

	// Verify formatting
	plainJoined := deltas[0].Format(true)
	if plainJoined != "[+] 192.168.1.1 came online (Online, 5ms)" {
		t.Errorf("unexpected plain joined string: %s", plainJoined)
	}
	ansiJoined := deltas[0].Format(false)
	if ansiJoined == plainJoined {
		t.Errorf("expected ANSI formatting codes in colored output: %s", ansiJoined)
	}

	plainDropped := deltas[1].Format(true)
	if plainDropped != "[-] 192.168.1.2 went offline" {
		t.Errorf("unexpected plain dropped string: %s", plainDropped)
	}
	ansiDropped := deltas[1].Format(false)
	if ansiDropped == plainDropped {
		t.Errorf("expected ANSI formatting codes in colored dropped output: %s", ansiDropped)
	}
}

