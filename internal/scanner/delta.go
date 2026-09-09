package scanner

import (
	"fmt"
	"net"
	"time"
)

// DeltaKind represents the category of change for an IP address between sweeps.
type DeltaKind string

const (
	DeltaNone    DeltaKind = ""
	DeltaJoined  DeltaKind = "joined"  // Host transitioned from offline to online/highlight/slow
	DeltaDropped DeltaKind = "dropped" // Host transitioned from active to offline
	DeltaChanged DeltaKind = "changed" // Host changed performance status (e.g. online -> slow)
)

// HostDelta describes a detected state change for an IP address across sweeps.
type HostDelta struct {
	IP   net.IP
	Old  HostStatus
	New  HostStatus
	RTT  time.Duration
	Kind DeltaKind
}

func (d HostDelta) String() string {
	return d.Format(true)
}

// Format returns a formatted string representation of the delta, with optional ANSI color codes.
func (d HostDelta) Format(plain bool) string {
	rttStr := FormatDurationMS(d.RTT)
	if plain {
		switch d.Kind {
		case DeltaJoined:
			return fmt.Sprintf("[+] %s came online (%s, %s)", d.IP, d.New, rttStr)
		case DeltaDropped:
			return fmt.Sprintf("[-] %s went offline", d.IP)
		case DeltaChanged:
			return fmt.Sprintf("[~] %s performance changed (%s -> %s, %s)", d.IP, d.Old, d.New, rttStr)
		default:
			return fmt.Sprintf("    %s unchanged", d.IP)
		}
	}

	switch d.Kind {
	case DeltaJoined:
		return fmt.Sprintf("\x1b[92;1m[+]\x1b[0m \x1b[1m%s\x1b[0m \x1b[32mcame online\x1b[0m (%s, %s)", d.IP, d.New, rttStr)
	case DeltaDropped:
		return fmt.Sprintf("\x1b[91;1m[-]\x1b[0m \x1b[1m%s\x1b[0m \x1b[31mwent offline\x1b[0m", d.IP)
	case DeltaChanged:
		return fmt.Sprintf("\x1b[93;1m[~]\x1b[0m \x1b[1m%s\x1b[0m \x1b[33mperformance changed\x1b[0m (%s -> %s, %s)", d.IP, d.Old, d.New, rttStr)
	default:
		return fmt.Sprintf("    %s unchanged", d.IP)
	}
}

// ComputeDeltas compares previous sweep results with current sweep results and returns detected changes.
func ComputeDeltas(prev, curr []HostResult) []HostDelta {
	if len(prev) == 0 || len(curr) == 0 {
		return nil
	}

	prevMap := make(map[string]HostResult, len(prev))
	for _, p := range prev {
		prevMap[p.IP.String()] = p
	}

	var deltas []HostDelta
	for _, c := range curr {
		p, exists := prevMap[c.IP.String()]
		if !exists {
			continue
		}

		if p.Status == StatusOffline && c.Status != StatusOffline {
			deltas = append(deltas, HostDelta{
				IP:   c.IP,
				Old:  p.Status,
				New:  c.Status,
				RTT:  c.RTT,
				Kind: DeltaJoined,
			})
		} else if p.Status != StatusOffline && c.Status == StatusOffline {
			deltas = append(deltas, HostDelta{
				IP:   c.IP,
				Old:  p.Status,
				New:  c.Status,
				RTT:  c.RTT,
				Kind: DeltaDropped,
			})
		} else if p.Status != StatusOffline && c.Status != StatusOffline && p.Status != c.Status {
			deltas = append(deltas, HostDelta{
				IP:   c.IP,
				Old:  p.Status,
				New:  c.Status,
				RTT:  c.RTT,
				Kind: DeltaChanged,
			})
		}
	}

	return deltas
}
