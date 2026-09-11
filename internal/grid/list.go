package grid

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// RenderList renders a clean, sorted tabular list of host names, IP addresses,
// round-trip ping times, and statuses. Active hosts are sorted by RTT ascending.
func RenderList(results []scanner.HostResult, isPlain bool, showOffline bool, totalScanned int, deltas []scanner.HostDelta) string {
	var active []scanner.HostResult
	var offline []scanner.HostResult

	for _, r := range results {
		if r.Status == scanner.StatusOffline {
			offline = append(offline, r)
		} else {
			active = append(active, r)
		}
	}

	// Sort active hosts by RTT ascending, then by IP address
	sort.Slice(active, func(i, j int) bool {
		if active[i].RTT != active[j].RTT {
			return active[i].RTT < active[j].RTT
		}
		return bytes.Compare(active[i].IP, active[j].IP) < 0
	})

	// Sort offline hosts by IP address ascending
	sort.Slice(offline, func(i, j int) bool {
		return bytes.Compare(offline[i].IP, offline[j].IP) < 0
	})

	var displayHosts []scanner.HostResult
	displayHosts = append(displayHosts, active...)
	if showOffline {
		displayHosts = append(displayHosts, offline...)
	}

	if len(displayHosts) == 0 {
		return fmt.Sprintf("No active hosts responded (out of %d scanned).\n", totalScanned)
	}

	type rowData struct {
		host      string
		ip        string
		rtt       string
		statusRaw string
		statusFmt string
		roleFmt   string
	}

	rows := make([]rowData, len(displayHosts))
	hostWidth := len("HOST")
	ipWidth := len("IP")
	rttWidth := len("RTT")
	var hasAnyRoles bool

	for i, r := range displayHosts {
		hName := r.Hostname
		if hName == "" {
			hName = r.IP.String()
		}
		ipStr := r.IP.String()

		rttStr := "-"
		if r.Status == scanner.StatusSilent {
			rttStr = "- (arp)"
		} else if r.Status != scanner.StatusOffline {
			rttStr = fmt.Sprintf("%.2fms", float64(r.RTT.Nanoseconds())/1e6)
		}

		statusRaw := strings.ToLower(r.Status.String())
		if r.Status == scanner.StatusHighlight {
			statusRaw = "fast"
		}

		statusFmt := statusRaw
		if !isPlain {
			switch r.Status {
			case scanner.StatusHighlight:
				statusFmt = "\033[92mfast\033[0m"
			case scanner.StatusOnline:
				statusFmt = "\033[96monline\033[0m"
			case scanner.StatusSlow:
				statusFmt = "\033[93mslow\033[0m"
			case scanner.StatusSilent:
				statusFmt = "\033[38;5;214msilent\033[0m"
			default:
				statusFmt = "\033[90moffline\033[0m"
			}
		}

		roleFmt := ""
		if len(r.Roles) > 0 {
			hasAnyRoles = true
			if isPlain {
				roleFmt = r.RoleBadge()
			} else {
				var coloredRoles []string
				for _, ro := range r.Roles {
					switch ro {
					case scanner.RoleLocalHost:
						coloredRoles = append(coloredRoles, "\033[96;1mMe\033[0m")
					case scanner.RoleGateway:
						coloredRoles = append(coloredRoles, "\033[92;1mGateway\033[0m")
					case scanner.RoleDNS:
						coloredRoles = append(coloredRoles, "\033[95;1mDNS\033[0m")
					case scanner.RoleDHCP:
						coloredRoles = append(coloredRoles, "\033[94;1mDHCP\033[0m")
					default:
						coloredRoles = append(coloredRoles, string(ro))
					}
				}
				roleFmt = "[" + strings.Join(coloredRoles, ", ") + "]"
			}
		}

		rows[i] = rowData{
			host:      hName,
			ip:        ipStr,
			rtt:       rttStr,
			statusRaw: statusRaw,
			statusFmt: statusFmt,
			roleFmt:   roleFmt,
		}

		if len(hName) > hostWidth {
			hostWidth = len(hName)
		}
		if len(ipStr) > ipWidth {
			ipWidth = len(ipStr)
		}
		if len(rttStr) > rttWidth {
			rttWidth = len(rttStr)
		}
	}

	var hasAnyMAC bool
	macWidth := len("MAC ADDRESS")
	vendorWidth := len("VENDOR")
	for _, r := range displayHosts {
		if r.MAC != "" {
			hasAnyMAC = true
			if len(r.MAC) > macWidth {
				macWidth = len(r.MAC)
			}
			if len(r.Vendor) > vendorWidth {
				vendorWidth = len(r.Vendor)
			}
		}
	}

	var b strings.Builder
	for i, row := range rows {
		r := displayHosts[i]
		prefix := ""
		if hasAnyRoles {
			prefix = "  "
			if row.roleFmt != "" {
				prefix = "+ "
			}
		}

		if i == 0 {
			// Print header
			headerPrefix := ""
			if hasAnyRoles {
				headerPrefix = "  "
			}
			b.WriteString(headerPrefix)
			fmt.Fprintf(&b, "%-*s  %-*s", hostWidth, "HOST", ipWidth, "IP")
			if hasAnyMAC {
				fmt.Fprintf(&b, "  %-*s  %-*s", macWidth, "MAC ADDRESS", vendorWidth, "VENDOR")
			}
			fmt.Fprintf(&b, "  %*s  %-8s", rttWidth, "RTT", "STATUS")
			if hasAnyRoles {
				b.WriteString("  ROLE")
			}
			b.WriteString("\n")
		}

		b.WriteString(prefix)
		fmt.Fprintf(&b, "%-*s  %-*s", hostWidth, row.host, ipWidth, row.ip)
		if hasAnyMAC {
			mStr := r.MAC
			if mStr == "" {
				mStr = "-"
			}
			vStr := r.Vendor
			if vStr == "" {
				vStr = "-"
			}
			fmt.Fprintf(&b, "  %-*s  %-*s", macWidth, mStr, vendorWidth, vStr)
		}
		fmt.Fprintf(&b, "  %*s  %-8s", rttWidth, row.rtt, row.statusFmt)
		if hasAnyRoles && row.roleFmt != "" {
			fmt.Fprintf(&b, "  %s", row.roleFmt)
		}
		b.WriteString("\n")
	}

	var joinedCount, droppedCount int
	for _, d := range deltas {
		switch d.Kind {
		case scanner.DeltaJoined:
			joinedCount++
		case scanner.DeltaDropped:
			droppedCount++
		}
	}

	if len(deltas) > 0 {
		fmt.Fprintf(&b, "\nTotal: %d active hosts (+%d joined, -%d dropped).\n", len(active), joinedCount, droppedCount)
	} else if totalScanned > 0 {
		fmt.Fprintf(&b, "\nTotal: %d active hosts (out of %d scanned).\n", len(active), totalScanned)
	} else {
		fmt.Fprintf(&b, "\nTotal: %d active hosts.\n", len(active))
	}

	return b.String()
}
