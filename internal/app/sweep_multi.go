package app

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/m8urnett/PingGrid/internal/grid"
	"github.com/m8urnett/PingGrid/internal/report"
	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/errors"
	"github.com/m8urnett/toolkit/log"
	"github.com/spf13/cobra"
)

type ifaceTask struct {
	info        scanner.InterfaceInfo
	ips         []net.IP
	scanCfg     scanner.Config
	gridCfg     grid.GridConfig
	prevResults []scanner.HostResult
}

type ifaceResult struct {
	task      ifaceTask
	results   []scanner.HostResult
	deltas    []scanner.HostDelta
	duration  time.Duration
	online    int
	highlight int
	slow      int
	silent    int
	offline   int
}

func runMultiInterfaceSweep(cmd *cobra.Command, flags *appFlags, baseGridCfg grid.GridConfig, logger *log.Logger) error {
	ifaces, err := scanner.GetActiveSweepableInterfaces()
	if err != nil {
		return errors.New(errors.ExitProcessing, "INTERFACE_DISCOVERY_FAILED", "Failed to discover active network interfaces", "", "Ensure at least one network adapter is up and has an assigned IPv4 address", err)
	}

	var tasks []ifaceTask
	for _, info := range ifaces {
		target := info.SweepCIDR()
		ips, autoGw, err := scanner.GenerateIPs(target, 0)
		if err != nil || len(ips) == 0 {
			continue
		}

		gw := info.Gateway
		if gw == nil && autoGw != nil {
			gw = autoGw
		}

		scanCfg := buildScannerConfig(cmd, flags, target, ips, gw, info.IP, info.DNSServers, info.DHCPServer)

		ifaceGridCfg := baseGridCfg
		ifaceGridCfg.InterfaceName = info.Name
		ifaceGridCfg.LinkHealth = &info.Health

		if !cmd.Flags().Changed("rows") && !cmd.Flags().Changed("cols") {
			ifaceGridCfg.Rows, ifaceGridCfg.Cols = grid.AutoLayout(len(ips))
		} else {
			ifaceGridCfg.Rows = flags.rows
			ifaceGridCfg.Cols = flags.cols
			if int64(len(ips)) > int64(ifaceGridCfg.Rows)*int64(ifaceGridCfg.Cols) {
				return errors.New(
					errors.ExitInput,
					"GRID_CAPACITY_EXCEEDED",
					fmt.Sprintf("Interface %s contains %d hosts, which exceeds grid capacity of %d cells", info.Name, len(ips), ifaceGridCfg.Rows*ifaceGridCfg.Cols),
					info.SweepCIDR(),
					"Increase --rows / --cols or omit them to automatically size each interface grid",
					nil,
				)
			}
		}

		autoW, autoH := grid.AutosizeDimensions(ifaceGridCfg.Rows, ifaceGridCfg.Cols, ifaceGridCfg.BorderWidth)
		if !cmd.Flags().Changed("width") {
			ifaceGridCfg.Width = autoW
		} else {
			ifaceGridCfg.Width = flags.width
		}
		if !cmd.Flags().Changed("height") {
			ifaceGridCfg.Height = autoH
		} else {
			ifaceGridCfg.Height = flags.height
		}
		if err := validateCanvasDimensions(ifaceGridCfg.Width, ifaceGridCfg.Height); err != nil {
			return errors.New(errors.ExitInput, "INPUT_TOO_LARGE", "Interface canvas exceeds the safety limit", info.Name, err.Error(), err)
		}

		tasks = append(tasks, ifaceTask{
			info:    info,
			ips:     ips,
			scanCfg: scanCfg,
			gridCfg: ifaceGridCfg,
		})
	}

	if len(tasks) == 0 {
		return errors.New(errors.ExitProcessing, "INTERFACE_SWEEP_FAILED", "No sweepable IP addresses generated from active interfaces", "", "Verify local interface subnet configurations", nil)
	}
	activeInterfaceScans := len(tasks)
	if activeInterfaceScans > flags.concurrency {
		activeInterfaceScans = flags.concurrency
	}
	workersPerInterface := flags.concurrency / activeInterfaceScans
	if workersPerInterface < 1 {
		workersPerInterface = 1
	}
	for i := range tasks {
		tasks[i].scanCfg.Concurrency = workersPerInterface
	}
	interfaceSlots := make(chan struct{}, activeInterfaceScans)

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	iteration := 0
	for {
		iteration++
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		resMap := make(map[int]ifaceResult)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for idx, task := range tasks {
			wg.Add(1)
			go func(i int, t ifaceTask) {
				defer wg.Done()
				select {
				case interfaceSlots <- struct{}{}:
					defer func() { <-interfaceSlots }()
				case <-ctx.Done():
					return
				}
				t0 := time.Now()
				results := scanner.Sweep(ctx, t.ips, t.scanCfg, nil)
				dur := time.Since(t0)
				deltas := scanner.ComputeDeltas(t.prevResults, results)

				on, high, sl, sil, off := countHostStatuses(results)

				res := ifaceResult{
					task:      t,
					results:   results,
					deltas:    deltas,
					duration:  dur,
					online:    on,
					highlight: high,
					slow:      sl,
					silent:    sil,
					offline:   off,
				}

				mu.Lock()
				resMap[i] = res
				mu.Unlock()
			}(idx, task)
		}

		wg.Wait()
		if ctx.Err() != nil {
			return nil
		}

		var ifaceResults []ifaceResult
		for idx := range tasks {
			ifaceResults = append(ifaceResults, resMap[idx])
			tasks[idx].prevResults = resMap[idx].results
		}

		var totalHosts, totalOnline, totalHigh, totalSlow, totalSilent, totalOffline int
		var maxDuration time.Duration
		for _, r := range ifaceResults {
			totalHosts += len(r.task.ips)
			totalOnline += (r.online + r.highlight + r.slow + r.silent)
			totalHigh += r.highlight
			totalSlow += r.slow
			totalSilent += r.silent
			totalOffline += r.offline
			if r.duration > maxDuration {
				maxDuration = r.duration
			}
		}

		isPlain := plainConsoleOutput(flags)

		switch flags.outputFormat {
		case "ascii":
			var asciiBuilder strings.Builder
			for _, r := range ifaceResults {
				if flags.showHUD {
					asciiBuilder.WriteString(r.task.info.Health.FormatBanner(r.task.info, isPlain))
				}
				primaryTag := ""
				if r.task.info.IsPrimary {
					primaryTag = ", Primary"
				}
				fmt.Fprintf(&asciiBuilder, "\n=== Interface: %s (%s%s) ===\n\n", r.task.info.Name, r.task.info.SweepCIDR(), primaryTag)
				gridOutput := grid.RenderASCII(r.task.gridCfg, r.results, isPlain)
				asciiBuilder.WriteString(gridOutput)
				asciiBuilder.WriteString("\n")
				silentStr := ""
				if r.silent > 0 {
					silentStr = fmt.Sprintf(" (%d silent/firewalled)", r.silent)
				}
				fmt.Fprintf(&asciiBuilder, "pg: %d/%d hosts active on %s%s\n\n", (r.online + r.highlight + r.slow + r.silent), len(r.task.ips), r.task.info.Name, silentStr)
			}
			var ifaceNames []string
			for _, r := range ifaceResults {
				ifaceNames = append(ifaceNames, r.task.info.Name)
			}
			fmt.Fprintf(&asciiBuilder, "pg: %d interfaces swept — %d/%d total hosts active across %s (%s)\n",
				len(ifaceResults), totalOnline, totalHosts, strings.Join(ifaceNames, ", "), scanner.FormatDurationMS(maxDuration))
			if flags.outputPath != "" {
				if err := writeOutputFile(flags.outputPath, []byte(asciiBuilder.String())); err != nil {
					return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save multi-interface ASCII grid", flags.outputPath, "Ensure directory is writable", err)
				}
				_ = logger.Diagnostic("pg: multi-interface ASCII grid saved to %s", flags.outputPath)
			} else {
				_ = logger.Data("%s", asciiBuilder.String())
			}

		case "list":
			var listBuilder strings.Builder
			for _, r := range ifaceResults {
				primaryTag := ""
				if r.task.info.IsPrimary {
					primaryTag = " [Primary]"
				}
				speedStr := r.task.info.Health.LinkSpeedStr
				if speedStr == "" {
					speedStr = "Unknown speed"
				}
				fmt.Fprintf(&listBuilder, "\n=== Interface: %s (%s%s) • %s ===\n", r.task.info.Name, r.task.info.SweepCIDR(), primaryTag, speedStr)
				resolved := scanner.ResolveHostnames(ctx, r.results, 200*time.Millisecond)
				listOut := grid.RenderList(resolved, isPlain, flags.Verbose, len(r.task.ips), r.deltas)
				listBuilder.WriteString(listOut + "\n")
			}
			if flags.outputPath != "" {
				if err := writeOutputFile(flags.outputPath, []byte(listBuilder.String())); err != nil {
					return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save host list", flags.outputPath, "Check directory permissions", err)
				}
				_ = logger.Diagnostic("pg: %d/%d total hosts active across %d interfaces. Saved to %s", totalOnline, totalHosts, len(ifaceResults), flags.outputPath)
			} else {
				_ = logger.Data("%s", listBuilder.String())
			}

		case "json":
			var ifaceEntries []report.Interface
			for _, r := range ifaceResults {
				active := r.online + r.highlight + r.slow + r.silent
				ipStr := ""
				if r.task.info.IP != nil {
					ipStr = r.task.info.IP.String()
				}
				gwStr := ""
				if r.task.info.Gateway != nil {
					gwStr = r.task.info.Gateway.String()
				}
				summary := report.Sweep{
					SchemaVersion: report.SchemaVersion,
					Target:        r.task.info.SweepCIDR(),
					ScanMode:      report.ScanMode(),
					TotalHosts:    len(r.task.ips),
					OnlineHosts:   active,
					FastHosts:     r.highlight,
					SlowHosts:     r.slow,
					SilentHosts:   r.silent,
					OfflineHosts:  r.offline,
					DurationMS:    r.duration.Milliseconds(),
					Rows:          r.task.gridCfg.Rows,
					Cols:          r.task.gridCfg.Cols,
				}
				ifaceEntries = append(ifaceEntries, report.Interface{
					Name:       r.task.info.Name,
					Index:      r.task.info.Index,
					Primary:    r.task.info.IsPrimary,
					Target:     r.task.info.SweepCIDR(),
					IP:         ipStr,
					Gateway:    gwStr,
					LinkHealth: &r.task.info.Health,
					Summary:    summary,
					Hosts:      report.Hosts(r.results),
				})
			}
			payload := report.MultiInterface{
				SchemaVersion:   report.SchemaVersion,
				MultiInterface:  true,
				InterfacesCount: len(ifaceResults),
				TotalHosts:      totalHosts,
				TotalActive:     totalOnline,
				TotalFast:       totalHigh,
				TotalSlow:       totalSlow,
				TotalSilent:     totalSilent,
				TotalOffline:    totalOffline,
				DurationMS:      maxDuration.Milliseconds(),
				Interfaces:      ifaceEntries,
			}
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			if flags.outputPath != "" {
				if err := writeOutputFile(flags.outputPath, data); err != nil {
					return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save JSON summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
				}
				_ = logger.Diagnostic("pg: %d/%d hosts active across %d interfaces. JSON saved to %s", totalOnline, totalHosts, len(ifaceResults), flags.outputPath)
			} else {
				_ = logger.Data("%s", string(data))
			}

		case "summary":
			var ifaceParts []string
			for _, r := range ifaceResults {
				ifaceParts = append(ifaceParts, fmt.Sprintf("%s: %d/%d active", r.task.info.Name, (r.online+r.highlight+r.slow+r.silent), len(r.task.ips)))
			}
			summaryLine := fmt.Sprintf("pg: %d interfaces swept (%s) in %s.", len(ifaceResults), strings.Join(ifaceParts, "; "), scanner.FormatDurationMS(maxDuration))
			if flags.outputPath != "" {
				if err := writeOutputFile(flags.outputPath, []byte(summaryLine+"\n")); err != nil {
					return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
				}
				_ = logger.Diagnostic("pg: multi-interface summary saved to %s", flags.outputPath)
			} else {
				_ = logger.Data("%s", summaryLine)
			}

		case "html":
			var cfgs []grid.GridConfig
			var resultsList [][]scanner.HostResult
			var deltasList [][]scanner.HostDelta
			var durations []time.Duration
			for _, r := range ifaceResults {
				cfgs = append(cfgs, r.task.gridCfg)
				resultsList = append(resultsList, r.results)
				deltasList = append(deltasList, r.deltas)
				durations = append(durations, r.duration)
			}
			refreshSec := int(flags.refresh.Seconds())
			htmlBytes, err := grid.RenderMultiHTMLWithDurations(cfgs, resultsList, durations, refreshSec, deltasList)
			if err != nil {
				return errors.New(errors.ExitProcessing, "RENDER_FAILED", "Failed to generate multi-interface HTML dashboard", "", "Check grid configuration", err)
			}
			outPath := flags.outputPath
			if outPath == "" {
				outPath = "grid.html"
			}
			if err := grid.SaveHTML(htmlBytes, outPath); err != nil {
				return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save multi-interface HTML dashboard", outPath, "Ensure directory is writable", err)
			}
			_ = logger.Diagnostic("pg: %d/%d hosts active across %d interfaces. Standalone HTML dashboard saved to %s", totalOnline, totalHosts, len(ifaceResults), outPath)

		case "iframe":
			var cfgs []grid.GridConfig
			var resultsList [][]scanner.HostResult
			var deltasList [][]scanner.HostDelta
			var durations []time.Duration
			for _, r := range ifaceResults {
				cfgs = append(cfgs, r.task.gridCfg)
				resultsList = append(resultsList, r.results)
				deltasList = append(deltasList, r.deltas)
				durations = append(durations, r.duration)
			}
			refreshSec := int(flags.refresh.Seconds())
			htmlBytes, err := grid.RenderMultiMinimalHTML(cfgs, resultsList, durations, refreshSec, deltasList)
			if err != nil {
				return errors.New(errors.ExitProcessing, "RENDER_FAILED", "Failed to generate multi-interface iframe HTML", "", "Check grid configuration", err)
			}
			outPath := flags.outputPath
			if outPath == "" {
				outPath = "grid-embed.html"
			}
			if err := grid.SaveHTML(htmlBytes, outPath); err != nil {
				return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save multi-interface iframe HTML", outPath, "Ensure directory is writable", err)
			}
			_ = logger.Diagnostic("pg: %d/%d hosts active across %d interfaces. Multi-interface iframe HTML saved to %s", totalOnline, totalHosts, len(ifaceResults), outPath)

		case "png":
			var cfgs []grid.GridConfig
			var resultsList [][]scanner.HostResult
			var compositeWidth, compositeHeight int64
			for _, r := range ifaceResults {
				cfgs = append(cfgs, r.task.gridCfg)
				resultsList = append(resultsList, r.results)
				if int64(r.task.gridCfg.Width) > compositeWidth {
					compositeWidth = int64(r.task.gridCfg.Width)
				}
				compositeHeight += int64(r.task.gridCfg.Height)
			}
			if compositeWidth*compositeHeight > maxCanvasPixels || compositeWidth > maxCanvasDimension || compositeHeight > maxCanvasDimension {
				return errors.New(errors.ExitInput, "INPUT_TOO_LARGE", "Composite multi-interface PNG exceeds the canvas safety limit", fmt.Sprintf("%dx%d", compositeWidth, compositeHeight), "Reduce dimensions or sweep fewer interfaces", nil)
			}
			img := grid.RenderMulti(cfgs, resultsList)
			outPath := flags.outputPath
			if outPath == "" {
				outPath = "grid.png"
			}
			if err := grid.SavePNG(img, outPath); err != nil {
				return errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save multi-grid PNG image", outPath, "Ensure directory is writable", err)
			}
			_ = logger.Diagnostic("pg: %d/%d hosts active across %d interfaces. Multi-interface grid image saved to %s", totalOnline, totalHosts, len(ifaceResults), outPath)
		}

		if flags.refresh <= 0 {
			break
		}

		_ = logger.Diagnostic("Sleeping for %v until next multi-interface sweep...", flags.refresh)
		select {
		case <-ctx.Done():
			_ = logger.Diagnostic("Multi-interface sweep loop terminated.")
			return nil
		case <-time.After(flags.refresh):
		}
	}

	return nil
}
