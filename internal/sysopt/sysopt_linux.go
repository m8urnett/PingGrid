//go:build linux

package sysopt

import (
	"bytes"
	"context"
	"fmt"
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

		out, err := writeSysctl(ctx, opt.ID, opt.TargetValue)
		if err != nil {
			sr.Err = fmt.Errorf("%v (%s)", err, out)
		} else {
			sr.Applied = true
			sr.Message = fmt.Sprintf("Set %s = %s", opt.ID, opt.TargetValue)
		}
		results = append(results, sr)
	}

	return results, nil
}

func readSysctl(ctx context.Context, key string) string {
	// Try reading directly from /proc/sys first
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
