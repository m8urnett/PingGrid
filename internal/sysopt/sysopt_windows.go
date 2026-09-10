//go:build windows

package sysopt

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var (
	modShell32        = syscall.NewLazyDLL("shell32.dll")
	procIsUserAnAdmin = modShell32.NewProc("IsUserAnAdmin")
)

// IsElevated returns true if the current process is running with Windows Administrator rights.
func IsElevated() bool {
	ret, _, _ := procIsUserAnAdmin.Call()
	return ret != 0
}

// Inspect gathers current OS network parameter values and proposed optimizations.
func Inspect(ctx context.Context) ([]Optimization, error) {
	var opts []Optimization

	// 1. Global Neighbor Cache Limit
	currLimit := getGlobalNeighborCacheLimit(ctx)
	opts = append(opts, Optimization{
		ID:           "win_neighbor_limit",
		Name:         "IPv4 Neighbor Cache Limit",
		Description:  "Expand global neighbor table capacity to prevent cache thrashing during large subnet sweeps (>256 hosts).",
		CurrentValue: currLimit,
		TargetValue:  "4096 entries per interface",
		Command:      "netsh interface ipv4 set global neighborcachelimit=4096",
	})

	// 2. Interface Reachable & Retransmit Timers
	ifaces := getActiveIPv4Interfaces()
	for _, iface := range ifaces {
		currReachable, currRetrans := getInterfaceTimers(ctx, iface.Index)
		opts = append(opts, Optimization{
			ID:           fmt.Sprintf("win_iface_timers_%d", iface.Index),
			Name:         fmt.Sprintf("Interface Reachable & Retransmit Timers (%s, index %d)", iface.Name, iface.Index),
			Description:  "Extend neighbor reachable cache duration to 5 mins and reduce dead-host retransmission delay to 200ms.",
			CurrentValue: fmt.Sprintf("BaseReachable: %s, Retransmit: %s", currReachable, currRetrans),
			TargetValue:  "BaseReachable: 300000 ms, Retransmit: 200 ms",
			Command:      fmt.Sprintf("netsh interface ipv4 set interface %d basereachable=300000 retransmit=200", iface.Index),
		})
	}

	// 3. Firewall ICMP Fastpath
	exePath, _ := os.Executable()
	fwStatus := getFirewallRuleStatus(ctx, "PingGrid ICMP Fastpath")
	opts = append(opts, Optimization{
		ID:           "win_firewall_fastpath",
		Name:         "Windows Firewall Outbound ICMP Fastpath",
		Description:  "Exempt PingGrid from outbound ICMP packet inspection queues in Windows Filtering Platform (WFP).",
		CurrentValue: fwStatus,
		TargetValue:  "Configured",
		Command:      fmt.Sprintf(`netsh advfirewall firewall add rule name="PingGrid ICMP Fastpath" dir=out action=allow program="%s" protocol=icmpv4`, exePath),
	})

	return opts, nil
}

// Apply executes the optimization adjustments, or simulates them if dryRun is true.
func Apply(ctx context.Context, dryRun bool) ([]StepResult, error) {
	opts, err := Inspect(ctx)
	if err != nil {
		return nil, err
	}

	var results []StepResult

	for _, opt := range opts {
		sr := StepResult{Opt: opt}

		if dryRun {
			sr.Skipped = true
			sr.Message = "Dry run (no changes applied)"
			results = append(results, sr)
			continue
		}

		if !IsElevated() {
			sr.Err = fmt.Errorf("administrative privileges required")
			results = append(results, sr)
			continue
		}

		switch opt.ID {
		case "win_neighbor_limit":
			if strings.Contains(opt.CurrentValue, "4096") {
				sr.Skipped = true
				sr.Message = "Already set to 4096 entries"
			} else {
				out, err := runNetsh(ctx, "interface", "ipv4", "set", "global", "neighborcachelimit=4096")
				if err != nil {
					sr.Err = fmt.Errorf("%v (%s)", err, out)
				} else {
					sr.Applied = true
					sr.Message = "Global neighbor cache limit expanded to 4096 entries"
				}
			}

		case "win_firewall_fastpath":
			if opt.CurrentValue == "Configured" {
				sr.Skipped = true
				sr.Message = "Firewall fastpath rule already exists"
			} else {
				exePath, _ := os.Executable()
				out, err := runNetsh(ctx, "advfirewall", "firewall", "add", "rule",
					`name=PingGrid ICMP Fastpath`,
					"dir=out",
					"action=allow",
					fmt.Sprintf("program=%s", exePath),
					"protocol=icmpv4",
				)
				if err != nil {
					sr.Err = fmt.Errorf("%v (%s)", err, out)
				} else {
					sr.Applied = true
					sr.Message = "Outbound ICMP fastpath rule registered in Windows Firewall"
				}
			}

		default:
			if strings.HasPrefix(opt.ID, "win_iface_timers_") {
				idxStr := strings.TrimPrefix(opt.ID, "win_iface_timers_")
				idx, _ := strconv.Atoi(idxStr)
				if idx > 0 {
					out, err := runNetsh(ctx, "interface", "ipv4", "set", "interface", strconv.Itoa(idx), "basereachable=300000", "retransmit=200")
					if err != nil {
						sr.Err = fmt.Errorf("%v (%s)", err, out)
					} else {
						sr.Applied = true
						sr.Message = fmt.Sprintf("Interface %d timers updated (Reachable: 300s, Retransmit: 200ms)", idx)
					}
				}
			}
		}

		results = append(results, sr)
	}

	return results, nil
}

func runNetsh(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "netsh", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func getGlobalNeighborCacheLimit(ctx context.Context) string {
	out, err := runNetsh(ctx, "interface", "ipv4", "show", "global")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(strings.ToLower(line), "neighbor cache limit") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Default (256)"
}

func getInterfaceTimers(ctx context.Context, ifIndex int) (string, string) {
	out, err := runNetsh(ctx, "interface", "ipv4", "show", "interface", strconv.Itoa(ifIndex))
	if err != nil {
		return "Unknown", "Unknown"
	}
	reachable := "Unknown"
	retransmit := "Unknown"
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "base reachable time") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				reachable = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(lower, "retransmission interval") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				retransmit = strings.TrimSpace(parts[1])
			}
		}
	}
	return reachable, retransmit
}

func getFirewallRuleStatus(ctx context.Context, ruleName string) string {
	out, err := runNetsh(ctx, "advfirewall", "firewall", "show", "rule", fmt.Sprintf("name=%s", ruleName))
	if err == nil && strings.Contains(out, ruleName) {
		return "Configured"
	}
	return "Not Configured"
}

func getActiveIPv4Interfaces() []net.Interface {
	var active []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return active
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasIPv4 := false
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() {
				hasIPv4 = true
				break
			}
		}
		if hasIPv4 {
			active = append(active, iface)
		}
	}
	return active
}
