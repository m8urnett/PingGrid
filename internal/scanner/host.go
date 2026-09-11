package scanner

import (
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	// MaxConcurrency bounds concurrent host workers to protect process and OS resources.
	MaxConcurrency = 1024
	// MaxPings bounds concurrent attempts made for a single host.
	MaxPings = 10
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
	// StatusSilent means the host was detected in the local ARP/neighbor cache but ICMP echo timed out (firewalled).
	StatusSilent
)

func (s HostStatus) String() string {
	switch s {
	case StatusOnline:
		return "Online"
	case StatusHighlight:
		return "Highlight"
	case StatusSlow:
		return "Slow"
	case StatusSilent:
		return "Silent"
	default:
		return "Offline"
	}
}

// HostRole identifies special network infrastructure roles for a host.
type HostRole string

const (
	RoleLocalHost HostRole = "Me"
	RoleGateway   HostRole = "Gateway"
	RoleDNS       HostRole = "DNS"
	RoleDHCP      HostRole = "DHCP"
)

// HostResult contains the ping sweep result for a single IP address.
type HostResult struct {
	IP       net.IP
	Hostname string
	Status   HostStatus
	Roles    []HostRole
	RTT      time.Duration
	MAC      string
	Vendor   string
	Err      error
}

// HasRole reports whether the host has the specified infrastructure role.
func (r HostResult) HasRole(role HostRole) bool {
	for _, ro := range r.Roles {
		if ro == role {
			return true
		}
	}
	return false
}

// RoleBadge returns a bracketed string representation of roles (e.g. "[Me]", "[Gateway, DNS]").
func (r HostResult) RoleBadge() string {
	if len(r.Roles) == 0 {
		return ""
	}
	var parts []string
	for _, ro := range r.Roles {
		parts = append(parts, string(ro))
	}
	return "[" + strings.Join(parts, ", ") + "]"
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
