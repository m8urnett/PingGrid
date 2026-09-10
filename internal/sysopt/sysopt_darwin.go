//go:build darwin

package sysopt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsElevated returns true if running as root on macOS.
func IsElevated() bool {
	return os.Geteuid() == 0
}

type darwinParam struct {
	key         string
	name        string
	description string
	target      string
}

var darwinParams = []darwinParam{
	{
		key:         "net.link.ether.inet.max_age",
		name:        "ARP Entry Maximum Age",
		description: "Extend retention lifespan of resolved ARP neighbor cache entries (seconds).",
		target:      "1200",
	},
	{
		key:         "net.link.ether.inet.prune_intvl",
		name:        "ARP Cache Prune Interval",
		description: "Configure how often expired ARP entries are cleaned (seconds).",
		target:      "60",
	},
	{
		key:         "kern.ipc.maxsockbuf",
		name:        "Maximum Socket Buffer Size",
		description: "Expand maximum socket buffer capacity for high-concurrency UDP/ICMP sockets.",
		target:      "4194304",
	},
}

// Inspect gathers current macOS sysctl values and proposed optimizations.
func Inspect(ctx context.Context) ([]Optimization, error) {
	var opts []Optimization
	for _, p := range darwinParams {
		curr := readDarwinSysctl(ctx, p.key)
		opts = append(opts, Optimization{
			ID:           p.key,
			Name:         p.name,
			Description:  p.description,
			CurrentValue: curr,
			TargetValue:  p.target,
			Command:      fmt.Sprintf("sysctl -w %s=%s", p.key, p.target),
		})
	}
	return opts, nil
}

// Apply executes the macOS sysctl adjustments, or simulates them if dryRun is true.
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

		out, err := writeDarwinSysctl(ctx, opt.ID, opt.TargetValue)
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

func readDarwinSysctl(ctx context.Context, key string) string {
	cmd := exec.CommandContext(ctx, "sysctl", "-n", key)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return strings.TrimSpace(out.String())
	}
	return "Unknown"
}

func writeDarwinSysctl(ctx context.Context, key, val string) (string, error) {
	cmd := exec.CommandContext(ctx, "sysctl", "-w", fmt.Sprintf("%s=%s", key, val))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}
