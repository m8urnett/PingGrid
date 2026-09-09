package scanner

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// HostStatus represents the ping status of a scanned IP address.
type HostStatus int

const (
	// StatusOffline means the host did not respond or timed out.
	StatusOffline HostStatus = iota
	// StatusOnline means the host responded within normal latency parameters.
	StatusOnline
	// StatusHighlight means the host is a designated gateway or infrastructure address.
	StatusHighlight
	// StatusSlow means the host responded but with latency exceeding the slow threshold.
	StatusSlow
)

func (s HostStatus) String() string {
	switch s {
	case StatusOnline:
		return "Online"
	case StatusHighlight:
		return "Highlight"
	case StatusSlow:
		return "Slow"
	default:
		return "Offline"
	}
}

// HostResult contains the ping sweep result for a single IP address.
type HostResult struct {
	IP       net.IP
	Hostname string
	Status   HostStatus
	RTT      time.Duration
	Err      error
}

// FormatDurationMS formats a duration as milliseconds (e.g. 5ms, 12.5ms, 0.45ms), never using µs or ns.
func FormatDurationMS(d time.Duration) string {
	if d <= 0 {
		return "0ms"
	}
	ms := float64(d.Nanoseconds()) / 1e6
	if ms == float64(int64(ms)) {
		return fmt.Sprintf("%dms", int64(ms))
	}
	if ms < 0.01 {
		return "<0.01ms"
	}
	s := fmt.Sprintf("%.2f", ms)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s + "ms"
}
