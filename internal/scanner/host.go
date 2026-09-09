package scanner

import (
	"net"
	"time"
)

// HostStatus represents the ping status of a scanned IP address.
type HostStatus int

const (
	// StatusOffline means the host did not respond or timed out.
	StatusOffline HostStatus = iota
	// StatusOnline means the host responded within normal latency parameters.
	StatusOnline
	// StatusHighlight means the host responded with ultra-low latency or is a gateway/highlight.
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
	IP     net.IP
	Status HostStatus
	RTT    time.Duration
	Err    error
}
