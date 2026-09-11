//go:build windows

package sysopt

import (
	"bytes"
	"context"
	"encoding/csv"
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

	// 3. Firewall ICMP permission
	exePath, _ := os.Executable()
	fwStatus := getFirewallRuleStatus(ctx, "PingGrid ICMP Fastpath")
	opts = append(opts, Optimization{
		ID:           "win_firewall_fastpath",
		Name:         "Windows Firewall Outbound ICMP Permission",
		Description:  "Ensure PingGrid is permitted to send outbound ICMP echo requests through Windows Firewall.",
		CurrentValue: fwStatus,
		TargetValue:  "Configured",
		Command:      fmt.Sprintf(`netsh advfirewall firewall add rule name="PingGrid ICMP Fastpath" dir=out action=allow program="%s" protocol=icmpv4`, exePath),
	})

	// 4. Global TCP Receive-Side Scaling (RSS)
	rssState := getGlobalTCPRSS(ctx)
	opts = append(opts, Optimization{
		ID:           "win_global_rss",
		Name:         "Global TCP Receive-Side Scaling (RSS)",
		Description:  "Enable multi-core kernel network packet processing to prevent single CPU bottlenecking during high-concurrency sweeps.",
		CurrentValue: rssState,
		TargetValue:  "enabled",
		Command:      "netsh int tcp set global rss=enabled",
	})

	// 5. Interface-Level Hardware Optimizations (RSS, EEE, Interrupt Moderation)
	nicProps := queryNICAdvancedProperties(ctx)
	for i, prop := range nicProps {
		lowerDisp := strings.ToLower(prop.displayName)
		if strings.Contains(lowerDisp, "receive side scaling") || strings.Contains(lowerDisp, "rss") {
			opts = append(opts, Optimization{
				ID:           fmt.Sprintf("win_nic_rss_%d", i),
				Name:         fmt.Sprintf("NIC Receive Side Scaling (%s)", prop.adapterName),
				Description:  "Distribute incoming packet receive processing across hardware queues and CPU cores to avoid NIC bottlenecking.",
				CurrentValue: prop.displayValue,
				TargetValue:  "Enabled",
				Command:      formatNICPropertyCommand(prop.adapterName, prop.displayName, "Enabled"),
				AdapterName:  prop.adapterName,
				PropertyName: prop.displayName,
			})
		} else if strings.Contains(lowerDisp, "energy efficient") || strings.Contains(lowerDisp, "eee") || strings.Contains(lowerDisp, "green") {
			opts = append(opts, Optimization{
				ID:           fmt.Sprintf("win_nic_eee_%d", i),
				Name:         fmt.Sprintf("NIC Energy Efficient Ethernet (%s)", prop.adapterName),
				Description:  "Disable transceiver Low Power Idle (LPI) sleep states to eliminate wake-up latency jitter and first-packet drops.",
				CurrentValue: prop.displayValue,
				TargetValue:  "Disabled",
				Command:      formatNICPropertyCommand(prop.adapterName, prop.displayName, "Disabled"),
				AdapterName:  prop.adapterName,
				PropertyName: prop.displayName,
			})
		} else if strings.Contains(lowerDisp, "interrupt moderation") {
			opts = append(opts, Optimization{
				ID:           fmt.Sprintf("win_nic_intmod_%d", i),
				Name:         fmt.Sprintf("NIC Interrupt Moderation (%s)", prop.adapterName),
				Description:  "Set hardware interrupt moderation to Adaptive for microsecond latency responses without packet loss.",
				CurrentValue: prop.displayValue,
				TargetValue:  "Adaptive",
				Command:      formatNICPropertyCommand(prop.adapterName, prop.displayName, "Adaptive"),
				AdapterName:  prop.adapterName,
				PropertyName: prop.displayName,
			})
		}
	}

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
			sr.DryRun = true
			sr.Message = "Dry run (no changes applied)"
			results = append(results, sr)
			continue
		}

		if !IsElevated() {
			sr.Err = fmt.Errorf("administrative privileges required")
			results = append(results, sr)
			continue
		}

		switch {
		case opt.ID == "win_neighbor_limit":
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

		case opt.ID == "win_firewall_fastpath":
			if opt.CurrentValue == "Configured" {
				sr.Skipped = true
				sr.Message = "Outbound ICMP permission rule already exists"
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
					sr.Message = "Outbound ICMP permission rule registered in Windows Firewall"
				}
			}

		case opt.ID == "win_global_rss":
			if strings.EqualFold(opt.CurrentValue, opt.TargetValue) {
				sr.Skipped = true
				sr.Message = "Global TCP RSS already enabled"
			} else {
				out, err := runNetsh(ctx, "int", "tcp", "set", "global", "rss=enabled")
				if err != nil {
					sr.Err = fmt.Errorf("%v (%s)", err, out)
				} else {
					sr.Applied = true
					sr.Message = "Global TCP Receive-Side Scaling enabled"
				}
			}

		case strings.HasPrefix(opt.ID, "win_nic_"):
			if strings.EqualFold(opt.CurrentValue, opt.TargetValue) {
				sr.Skipped = true
				sr.Message = fmt.Sprintf("Already set to %s", opt.TargetValue)
			} else {
				out, err := setNICAdvancedProperty(ctx, opt.AdapterName, opt.PropertyName, opt.TargetValue)
				if err != nil {
					sr.Err = fmt.Errorf("%v (%s)", err, out)
				} else {
					sr.Applied = true
					sr.Message = fmt.Sprintf("%s updated to %s", opt.Name, opt.TargetValue)
				}
			}

		case strings.HasPrefix(opt.ID, "win_iface_timers_"):
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

func getGlobalTCPRSS(ctx context.Context) string {
	out, err := runNetsh(ctx, "int", "tcp", "show", "global")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "receive-side scaling state") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

type nicAdvProperty struct {
	adapterName  string
	displayName  string
	displayValue string
}

func queryNICAdvancedProperties(ctx context.Context) []nicAdvProperty {
	script := `Get-NetAdapterAdvancedProperty -DisplayName '*Receive Side Scaling*','*Energy Efficient*','*Interrupt Moderation*','*EEE*','*Green*' -ErrorAction SilentlyContinue | Select-Object Name, DisplayName, DisplayValue | ConvertTo-Csv -NoTypeInformation`
	out, err := runPowerShell(ctx, script)
	if err != nil || len(out) == 0 {
		return nil
	}

	headerIdx := strings.Index(out, `"Name"`)
	if headerIdx == -1 {
		return nil
	}

	reader := csv.NewReader(strings.NewReader(out[headerIdx:]))
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil
	}

	var props []nicAdvProperty
	for _, rec := range records[1:] {
		if len(rec) >= 3 {
			props = append(props, nicAdvProperty{
				adapterName:  strings.TrimSpace(rec[0]),
				displayName:  strings.TrimSpace(rec[1]),
				displayValue: strings.TrimSpace(rec[2]),
			})
		}
	}
	return props
}

func runPowerShell(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(stderr.String()), err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func formatNICPropertyCommand(adapterName, propertyName, value string) string {
	escape := func(value string) string { return strings.ReplaceAll(value, "'", "''") }
	return fmt.Sprintf(
		"powershell -NoProfile -Command \"Set-NetAdapterAdvancedProperty -Name '%s' -DisplayName '%s' -DisplayValue '%s'\"",
		escape(adapterName), escape(propertyName), escape(value),
	)
}

func setNICAdvancedProperty(ctx context.Context, adapterName, propertyName, value string) (string, error) {
	const script = `Set-NetAdapterAdvancedProperty -Name $env:PINGGRID_ADAPTER_NAME -DisplayName $env:PINGGRID_PROPERTY_NAME -DisplayValue $env:PINGGRID_PROPERTY_VALUE -ErrorAction Stop`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Env = append(os.Environ(),
		"PINGGRID_ADAPTER_NAME="+adapterName,
		"PINGGRID_PROPERTY_NAME="+propertyName,
		"PINGGRID_PROPERTY_VALUE="+value,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(stderr.String()), err
	}
	return strings.TrimSpace(stdout.String()), nil
}
