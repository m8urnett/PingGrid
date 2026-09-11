//go:build linux

package sysopt

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

// IsElevated returns true if running as root on Linux.
func IsElevated() bool {
	return os.Geteuid() == 0
}

type sysctlParam struct {
	key         string
	name        string
	description string
	target      string
}

var linuxParams = []sysctlParam{
	{
		key:         "net.ipv4.neigh.default.mcast_solicit",
		name:        "ARP Multicast Solicitations",
		description: "Reduce dead-host ARP query probes from 3 attempts to 1.",
		target:      "1",
	},
	{
		key:         "net.ipv4.neigh.default.retrans_time_ms",
		name:        "ARP Retransmission Delay",
		description: "Lower ARP retransmission wait time from 1000ms to 100ms.",
		target:      "100",
	},
	{
		key:         "net.ipv4.neigh.default.base_reachable_time_ms",
		name:        "Neighbor Reachable Duration",
		description: "Extend neighbor cache retention duration to 5 minutes.",
		target:      "300000",
	},
	{
		key:         "net.ipv4.neigh.default.gc_thresh3",
		name:        "Neighbor Cache Limit (gc_thresh3)",
		description: "Expand maximum neighbor cache table size to 4096 entries.",
		target:      "4096",
	},
	{
		key:         "net.ipv4.ping_group_range",
		name:        "Unprivileged ICMP Ping Socket Range",
		description: "Allow unprivileged group accounts to create raw/datagram ICMP sockets.",
		target:      "0 2147483647",
	},
}

// Inspect gathers current Linux sysctl values and proposed optimizations.
func Inspect(ctx context.Context) ([]Optimization, error) {
	var opts []Optimization
	for _, p := range linuxParams {
		curr := readSysctl(ctx, p.key)
		opts = append(opts, Optimization{
			ID:           p.key,
			Name:         p.name,
			Description:  p.description,
			CurrentValue: curr,
			TargetValue:  p.target,
			Command:      fmt.Sprintf("sysctl -w %s=\"%s\"", p.key, p.target),
		})
	}

	// Interface-level optimizations (EEE, Interrupt Coalescing) via ethtool
	ifaces := getActiveIPv4Interfaces()
	for _, iface := range ifaces {
		// EEE (Energy Efficient Ethernet)
		eeeStatus := readEthtoolEEE(ctx, iface.Name)
		if eeeStatus != "" {
			opts = append(opts, Optimization{
				ID:           fmt.Sprintf("linux_eee_%s", iface.Name),
				Name:         fmt.Sprintf("NIC Energy Efficient Ethernet (%s)", iface.Name),
				Description:  "Disable transceiver Low Power Idle (LPI) sleep states to eliminate wake-up latency jitter and packet drop.",
				CurrentValue: eeeStatus,
				TargetValue:  "Disabled",
				Command:      fmt.Sprintf("ethtool --set-eee %s eee off", iface.Name),
			})
		}

		// Interrupt Coalescing
		coalesceStatus := readEthtoolCoalesce(ctx, iface.Name)
		if coalesceStatus != "" {
			opts = append(opts, Optimization{
				ID:           fmt.Sprintf("linux_coalesce_%s", iface.Name),
				Name:         fmt.Sprintf("NIC Adaptive Interrupt Coalescing (%s)", iface.Name),
				Description:  "Enable dynamic hardware interrupt moderation for microsecond latency responses without packet loss.",
				CurrentValue: coalesceStatus,
				TargetValue:  "Adaptive RX: on",
				Command:      fmt.Sprintf("ethtool -C %s adaptive-rx on", iface.Name),
			})
		}
	}

	return opts, nil
}

// Apply executes the Linux sysctl adjustments, or simulates them if dryRun is true.
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
			sr.DryRun = true
			sr.Message = "Dry run (no changes applied)"
			results = append(results, sr)
			continue
		}

		if !IsElevated() {
			sr.Err = fmt.Errorf("root privileges required")
			results = append(results, sr)
			continue
		}

		if strings.TrimSpace(opt.CurrentValue) == strings.TrimSpace(opt.TargetValue) {
			sr.Skipped = true
			sr.Message = fmt.Sprintf("Already set to %s", opt.TargetValue)
			results = append(results, sr)
			continue
		}

		switch {
		case strings.HasPrefix(opt.ID, "linux_eee_"):
			ifaceName := strings.TrimPrefix(opt.ID, "linux_eee_")
			out, err := runCommand(ctx, "ethtool", "--set-eee", ifaceName, "eee", "off")
			if err != nil {
				sr.Err = fmt.Errorf("%v (%s)", err, out)
			} else {
				sr.Applied = true
				sr.Message = fmt.Sprintf("Disabled EEE on %s", ifaceName)
			}
		case strings.HasPrefix(opt.ID, "linux_coalesce_"):
			ifaceName := strings.TrimPrefix(opt.ID, "linux_coalesce_")
			out, err := runCommand(ctx, "ethtool", "-C", ifaceName, "adaptive-rx", "on")
			if err != nil {
				sr.Err = fmt.Errorf("%v (%s)", err, out)
			} else {
				sr.Applied = true
				sr.Message = fmt.Sprintf("Enabled Adaptive RX coalescing on %s", ifaceName)
			}
		default:
			out, err := writeSysctl(ctx, opt.ID, opt.TargetValue)
			if err != nil {
				sr.Err = fmt.Errorf("%v (%s)", err, out)
			} else {
				sr.Applied = true
				sr.Message = fmt.Sprintf("Set %s = %s", opt.ID, opt.TargetValue)
			}
		}

		results = append(results, sr)
	}

	return results, nil
}

func readSysctl(ctx context.Context, key string) string {
	procPath := "/proc/sys/" + strings.ReplaceAll(key, ".", "/")
	if data, err := os.ReadFile(procPath); err == nil {
		return strings.TrimSpace(string(data))
	}

	cmd := exec.CommandContext(ctx, "sysctl", "-n", key)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return strings.TrimSpace(out.String())
	}
	return "Unknown"
}

func writeSysctl(ctx context.Context, key, val string) (string, error) {
	cmd := exec.CommandContext(ctx, "sysctl", "-w", fmt.Sprintf("%s=%s", key, val))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func readEthtoolEEE(ctx context.Context, iface string) string {
	out, err := runCommand(ctx, "ethtool", "--show-eee", iface)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EEE status:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				val := strings.TrimSpace(parts[1])
				if strings.Contains(strings.ToLower(val), "enabled") {
					return "Enabled"
				}
				return "Disabled"
			}
		}
	}
	return ""
}

func readEthtoolCoalesce(ctx context.Context, iface string) string {
	out, err := runCommand(ctx, "ethtool", "-c", iface)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Adaptive RX:") {
			return line
		}
	}
	return ""
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
