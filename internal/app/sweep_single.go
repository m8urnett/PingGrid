package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/m8urnett/PingGrid/internal/grid"
	"github.com/m8urnett/PingGrid/internal/report"
	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/errors"
	"github.com/m8urnett/toolkit/log"
	"github.com/spf13/cobra"
)

func runSweep(cmd *cobra.Command, flags *appFlags, args []string) error {
	// Positional target argument support (e.g. pg 10.8.0.1/24)
	if len(args) > 0 {
		flags.target = args[0]
	}
	if err := validateScanFlags(flags); err != nil {
		return err
	}

	logger := log.New(flags.Quiet)

	gridCfg, err := buildGridConfig(cmd, flags)
	if err != nil {
		return err
	}

	if err := resolveOutputOptions(cmd, flags); err != nil {
		return err
	}

	if flags.allInterfaces {
		if flags.ifaceTarget != "" {
			return errors.New(errors.ExitInput, "USAGE_CONFLICTING_FLAGS", "Conflicting interface options specified", "-i/--interface and -A/--all-interfaces", "Specify either a single interface with -i or sweep all with -A/--all-interfaces", nil)
		}
		if flags.target != "" {
			return errors.New(errors.ExitInput, "USAGE_CONFLICTING_FLAGS", "Conflicting target and all-interfaces options specified", flags.target, "Omit positional target when sweeping all interfaces with -A/--all-interfaces", nil)
		}
		return runMultiInterfaceSweep(cmd, flags, gridCfg, logger)
	}

	resolved, err := resolveScanTarget(flags)
	if err != nil {
		return err
	}
	ips := resolved.ips
	activeIface := resolved.activeIface
	gwIP := resolved.gateway
	localHostIP := resolved.localHost
	dnsServers := resolved.dnsServers
	dhcpServer := resolved.dhcpServer

	rowsChanged := cmd.Flags().Changed("rows")
	colsChanged := cmd.Flags().Changed("cols")

	switch {
	case !rowsChanged && !colsChanged:
		// Automatically size grid rows and columns based on total hosts in target IP range
		flags.rows, flags.cols = grid.AutoLayout(len(ips))

	case colsChanged && !rowsChanged:
		// User specified columns; adapt rows to fit range
		flags.rows = (len(ips) + flags.cols - 1) / flags.cols
		if flags.rows < 1 {
			flags.rows = 1
		}

	case rowsChanged && !colsChanged:
		// User specified rows; adapt columns to fit range
		flags.cols = (len(ips) + flags.rows - 1) / flags.rows
		if flags.cols < 1 {
			flags.cols = 1
		}

	case rowsChanged && colsChanged:
		// Both rows and columns explicitly specified by user: verify capacity
		totalSlots := flags.rows * flags.cols
		if len(ips) > totalSlots {
			return errors.New(
				errors.ExitInput,
				"GRID_CAPACITY_EXCEEDED",
				fmt.Sprintf("Target IP range contains %d hosts, which exceeds grid capacity of %d cells (%d rows x %d cols)", len(ips), totalSlots, flags.rows, flags.cols),
				fmt.Sprintf("%d hosts > %d slots", len(ips), totalSlots),
				"Increase --rows / --cols or omit them to automatically size the grid to fit the IP range",
				nil,
			)
		}
	}

	// Autosize canvas width and height to preserve square 8x8 cell size and proportions
	autoW, autoH := grid.AutosizeDimensions(flags.rows, flags.cols, flags.borderWidth)
	if !cmd.Flags().Changed("width") {
		flags.width = autoW
	}
	if !cmd.Flags().Changed("height") {
		flags.height = autoH
	}
	if err := validateCanvasDimensions(flags.width, flags.height); err != nil {
		return err
	}

	gridCfg.Rows = flags.rows
	gridCfg.Cols = flags.cols
	gridCfg.Width = flags.width
	gridCfg.Height = flags.height
	gridCfg.BorderWidth = flags.borderWidth
	if activeIface != nil {
		gridCfg.LinkHealth = &activeIface.Health
		gridCfg.InterfaceName = activeIface.Name
	}

	scanCfg := buildScannerConfig(cmd, flags, flags.target, ips, gwIP, localHostIP, dnsServers, dhcpServer)

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if flags.Verbose {
		type ipRoleEntry struct {
			ip    string
			roles []string
		}
		var entries []ipRoleEntry
		addRole := func(ip net.IP, role string) {
			if ip == nil {
				return
			}
			ipStr := ip.String()
			for i := range entries {
				if entries[i].ip == ipStr {
					for _, r := range entries[i].roles {
						if r == role {
							return
						}
					}
					entries[i].roles = append(entries[i].roles, role)
					return
				}
			}
			entries = append(entries, ipRoleEntry{ip: ipStr, roles: []string{role}})
		}

		if localHostIP != nil {
			addRole(localHostIP, "Me")
		}
		if gwIP != nil {
			addRole(gwIP, "Gateway")
		}
		for _, dns := range dnsServers {
			addRole(dns, "DNS")
		}
		if dhcpServer != nil {
			addRole(dhcpServer, "DHCP")
		}

		if len(entries) > 0 {
			_ = logger.Diagnostic("Discovered network topology hosts:")
			for _, entry := range entries {
				_ = logger.Diagnostic("  + %-15s [%s]", entry.ip, strings.Join(entry.roles, ", "))
			}
		}
	}

	var prevResults []scanner.HostResult
	iteration := 0
	for {
		iteration++
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		currResults, err := executeSweepIteration(ctx, flags, gridCfg, ips, scanCfg, logger, iteration, prevResults)
		if err != nil {
			return err
		}
		prevResults = currResults

		if flags.refresh <= 0 {
			break
		}

		_ = logger.Diagnostic("Sleeping for %v until next sweep...", flags.refresh)
		select {
		case <-ctx.Done():
			_ = logger.Diagnostic("Sweep loop terminated.")
			return nil
		case <-time.After(flags.refresh):
		}
	}

	return nil
}

func executeSweepIteration(
	ctx context.Context,
	flags *appFlags,
	gridCfg grid.GridConfig,
	ips []net.IP,
	scanCfg scanner.Config,
	logger *log.Logger,
	iteration int,
	prevResults []scanner.HostResult,
) ([]scanner.HostResult, error) {
	_ = logger.Diagnostic("Starting ping sweep of %d addresses (target: %s, concurrency: %d, cycle: #%d)...",
		len(ips), flags.target, flags.concurrency, iteration)

	start := time.Now()
	results := scanner.Sweep(ctx, ips, scanCfg, func(completed, total int, res scanner.HostResult) {
		if flags.Verbose {
			if len(res.Roles) > 0 {
				roleBadge := res.RoleBadge()
				if res.Status != scanner.StatusOffline {
					_ = logger.Diagnostic("+ Host %s %s responded in %s (%s)", res.IP, roleBadge, scanner.FormatDurationMS(res.RTT), res.Status)
				} else {
					_ = logger.Diagnostic("+ Host %s %s did not respond (offline)", res.IP, roleBadge)
				}
			} else if res.Status != scanner.StatusOffline {
				_ = logger.Diagnostic("  Host %s responded in %s (%s)", res.IP, scanner.FormatDurationMS(res.RTT), res.Status)
			}
		}
	})
	duration := time.Since(start)

	onlineCount, highlightCount, slowCount, silentCount, offlineCount := countHostStatuses(results)

	activeCount := onlineCount + highlightCount + slowCount + silentCount
	deltas := scanner.ComputeDeltas(prevResults, results)

	if flags.Verbose && silentCount > 0 {
		for _, r := range results {
			if r.Status == scanner.StatusSilent {
				vendorStr := ""
				if r.Vendor != "" {
					vendorStr = fmt.Sprintf(" [%s]", r.Vendor)
				}
				_ = logger.Diagnostic("? Host %s (MAC: %s)%s active via ARP (ICMP echo blocked)", r.IP, r.MAC, vendorStr)
			}
		}
	}

	var joinedHosts, droppedHosts []string
	for _, d := range deltas {
		switch d.Kind {
		case scanner.DeltaJoined:
			joinedHosts = append(joinedHosts, d.IP.String())
		case scanner.DeltaDropped:
			droppedHosts = append(droppedHosts, d.IP.String())
		}
	}

	_ = logger.Diagnostic("Sweep completed in %s. Active: %d (Fast: %d, Normal: %d, Slow: %d, Silent: %d), Offline: %d, Deltas: %d",
		scanner.FormatDurationMS(duration), activeCount, highlightCount, onlineCount, slowCount, silentCount, offlineCount, len(deltas))

	// Save to file if output path is configured
	if flags.outputPath != "" {
		if flags.outputFormat == "png" || strings.HasSuffix(strings.ToLower(flags.outputPath), ".png") {
			img := grid.Render(gridCfg, results)
			if err := grid.SavePNG(img, flags.outputPath); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save PNG image", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			_ = logger.Diagnostic("Grid saved to %s (%dx%d, %d rows x %d cols)",
				flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
		} else if flags.outputFormat == "iframe" {
			refreshSec := int(flags.refresh.Seconds())
			minBytes, err := grid.RenderMinimalHTML(gridCfg, results, duration, refreshSec, deltas)
			if err != nil {
				return nil, errors.New(errors.ExitProcessing, "RENDER_FAILED", "Failed to generate minimal embed HTML", "", "Check grid configuration parameters", err)
			}
			if err := grid.SaveHTML(minBytes, flags.outputPath); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save minimal embed HTML file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			_ = logger.Diagnostic("Embeddable iframe HTML saved to %s (%dx%d, %d rows x %d cols)",
				flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
		} else if flags.outputFormat == "html" {
			refreshSec := int(flags.refresh.Seconds())
			htmlBytes, err := grid.RenderHTML(gridCfg, results, duration, refreshSec, deltas)
			if err != nil {
				return nil, errors.New(errors.ExitProcessing, "RENDER_FAILED", "Failed to generate standalone HTML dashboard", "", "Check grid configuration parameters", err)
			}
			if err := grid.SaveHTML(htmlBytes, flags.outputPath); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save standalone HTML file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			_ = logger.Diagnostic("Standalone HTML dashboard saved to %s (%dx%d, %d rows x %d cols)",
				flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
		}
	}

	// Console Output Rendering
	isPlain := plainConsoleOutput(flags)

	switch flags.outputFormat {
	case "png":
		if !flags.Quiet {
			_ = logger.Diagnostic("pg: %d/%d hosts active. Activity grid image saved to %s", activeCount, len(ips), flags.outputPath)
		}

	case "json":
		summary := report.Sweep{
			SchemaVersion: report.SchemaVersion,
			Target:        flags.target,
			ScanMode:      report.ScanMode(),
			TotalHosts:    len(ips),
			OnlineHosts:   activeCount,
			FastHosts:     highlightCount,
			SlowHosts:     slowCount,
			SilentHosts:   silentCount,
			OfflineHosts:  offlineCount,
			DurationMS:    duration.Milliseconds(),
			OutputFile:    flags.outputPath,
			Width:         flags.width,
			Height:        flags.height,
			Rows:          flags.rows,
			Cols:          flags.cols,
			JoinedHosts:   joinedHosts,
			DroppedHosts:  droppedHosts,
			Hosts:         report.Hosts(results),
		}
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return nil, err
		}
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, data); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save JSON summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Diagnostic("pg: %d/%d hosts active. JSON summary saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else {
			_ = logger.Data("%s", string(data))
		}

	case "summary":
		deltaSummary := ""
		if len(deltas) > 0 {
			deltaSummary = fmt.Sprintf(" (+%d joined, -%d dropped)", len(joinedHosts), len(droppedHosts))
		}
		silentSummary := ""
		if silentCount > 0 {
			silentSummary = fmt.Sprintf(" (%d silent/firewalled)", silentCount)
		}
		summaryLine := fmt.Sprintf("%d/%d hosts active%s%s. Output: %s",
			activeCount, len(ips), silentSummary, deltaSummary, flags.outputPath)
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, []byte(summaryLine+"\n")); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Diagnostic("pg: %d/%d hosts active. Summary saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else if !flags.Quiet {
			_ = logger.Data("%s", summaryLine)
		}

	case "html":
		if !flags.Quiet {
			if flags.outputPath != "" {
				_ = logger.Diagnostic("pg: %d/%d hosts active. Standalone HTML dashboard saved to %s", activeCount, len(ips), flags.outputPath)
			}
		}

	case "iframe":
		if !flags.Quiet {
			if flags.outputPath != "" {
				_ = logger.Diagnostic("pg: %d/%d hosts active. Embeddable iframe HTML saved to %s", activeCount, len(ips), flags.outputPath)
			}
		}

	case "list":
		resolved := scanner.ResolveHostnames(ctx, results, 200*time.Millisecond)
		if flags.outputPath != "" {
			listText := grid.RenderList(resolved, true, flags.Verbose, len(ips), deltas)
			if err := writeOutputFile(flags.outputPath, []byte(listText)); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save host list file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Diagnostic("pg: %d/%d hosts active. Host list saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else if !flags.Quiet {
			listText := grid.RenderList(resolved, isPlain, flags.Verbose, len(ips), deltas)
			if iteration > 1 {
				fmt.Printf("\n--- pg Sweep #%d (%s) ---\n", iteration, time.Now().Format("15:04:05"))
			}
			fmt.Print(listText)
		}

	default: // "ascii"
		if flags.showHUD && gridCfg.LinkHealth != nil && iteration == 1 && flags.outputPath == "" && !flags.Quiet {
			hudBanner := gridCfg.LinkHealth.FormatBanner(scanner.InterfaceInfo{
				Name:       gridCfg.InterfaceName,
				IP:         scanCfg.LocalHostIP,
				Gateway:    scanCfg.GatewayIP,
				DNSServers: scanCfg.DNSServers,
				Health:     *gridCfg.LinkHealth,
			}, isPlain)
			fmt.Print(hudBanner)
		}
		asciiGrid := grid.RenderASCIIDelta(gridCfg, results, deltas, isPlain)
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, []byte(asciiGrid)); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save ASCII grid file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Diagnostic("pg: %d/%d hosts active. Visual grid saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else if !flags.Quiet {
			if iteration > 1 {
				fmt.Printf("\n--- pg Sweep #%d (%s) ---\n", iteration, time.Now().Format("15:04:05"))
				if len(deltas) > 0 {
					fmt.Printf("State Changes (%d):\n", len(deltas))
					for _, d := range deltas {
						fmt.Printf("  %s\n", d.Format(isPlain))
					}
				} else {
					fmt.Println("State Changes: None (all hosts stable)")
				}
			}
			fmt.Print(asciiGrid)
			if silentCount > 0 {
				_ = logger.Data("pg: %d/%d hosts active (%d silent/firewalled).", activeCount, len(ips), silentCount)
			} else {
				_ = logger.Data("pg: %d/%d hosts active.", activeCount, len(ips))
			}
		}
	}

	return results, nil
}
