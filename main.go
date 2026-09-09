package main

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
	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/PingGrid/internal/toolkit/cli"
	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
	"github.com/m8urnett/PingGrid/internal/toolkit/log"
	"github.com/m8urnett/PingGrid/internal/toolkit/paths"
	"github.com/m8urnett/PingGrid/internal/version"
	"github.com/spf13/cobra"
)

type appFlags struct {
	cli.StandardFlags

	scheme         string
	target         string
	rows           int
	cols           int
	width          int
	height         int
	borderWidth    int
	outputPath     string
	colorOffline   string
	colorOnline    string
	colorHighlight string
	colorSlow      string
	colorBorder    string
	colorFrame     string
	concurrency    int
	fastThreshold  time.Duration
	slowThreshold  time.Duration
	gateway        string
	refresh        time.Duration
	showVersion    bool

	// Dedicated output format flags
	optText    bool
	optJSON    bool
	optSummary bool
	optHTML    bool
	optIframe  bool
}

func main() {
	// Support Windows standard help and version switches (/?, /h, /help, -?, /v, /version)
	for i := 1; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "/?", "-?", "/h", "/help":
			os.Args[i] = "--help"
		case "/v", "/version":
			os.Args[i] = "--version"
		}
	}

	rootCmd, _ := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		errors.Fatal(err)
	}
}

func newRootCmd() (*cobra.Command, *appFlags) {
	var flags appFlags

	rootCmd := &cobra.Command{
		Use:   "pg [target]",
		Short: "Ping sweep subnet visualizer and grid image generator",
		Long: `pg (PingGrid) performs a concurrent ping sweep across a target subnet or IP range
and renders an activity grid (terminal ASCII display, interactive HTML, or PNG image).`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(os.Args) <= 1 {
				_ = cmd.Help()
				return nil
			}
			if flags.showVersion {
				fmt.Println(version.Full())
				return nil
			}
			return runSweep(cmd, &flags, args)
		},
	}

	rootCmd.Example = `  # Sweep target subnet (terminal ASCII grid output):
  pg 10.8.0.1/24

  # Continuous live monitoring: refresh sweep every 5 seconds with delta tracking:
  pg 192.168.1.0/24 -R 5s

  # Standalone interactive HTML dashboard (saves to grid.html):
  pg 10.8.0.1/24 --html

  # Standalone HTML dashboard with custom filename:
  pg 10.8.0.1/24 --html -f dashboard.html

  # Minimal embeddable HTML for iframes (saves to grid-embed.html):
  pg 10.8.0.1/24 --iframe -f embed.html

  # Output as machine-readable JSON:
  pg 10.8.0.1/24 --json

  # Output single-line summary:
  pg 10.8.0.1/24 --summary

  # Plain monochrome ASCII mode for simple terminals or automation:
  pg 10.8.0.1/24 --plain

  # Output as PowerShell-friendly JSON summary:
  pg 10.8.0.1/24 --ps`

	helpTemplate := `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}Usage:
  {{.UseLine}}

{{if .HasAvailableLocalFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

{{end}}{{if .HasAvailableInheritedFlags}}Standard Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}

{{end}}{{if .HasExample}}Examples:
{{.Example}}

{{end}}Shell Completion:
  Generate tab autocompletion script for commands, flags, and color schemes.

  PowerShell (load in current session):
    pg completion powershell | Out-String | Invoke-Expression

  PowerShell (persist across all sessions in $PROFILE):
    Add-Content $PROFILE "` + "`" + `\npg completion powershell | Out-String | Invoke-Expression` + "`" + `"

  Completes all flags (-s/--scheme, --text, --json, --html, --iframe, --summary, -o/--output, -R/--refresh, etc.), built-in
  color schemes (dark, light, earth, moss, linen), and output formats.
`
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	cli.BindStandardFlags(rootCmd, &flags.StandardFlags)
	if flag := rootCmd.PersistentFlags().Lookup("output"); flag != nil {
		flag.Usage = "Output format: text, json, summary, html (standalone dashboard), iframe (embeddable minimal)"
	}

	rootCmd.Flags().BoolVar(&flags.optText, "text", false, "Output console ASCII text grid (default)")
	rootCmd.Flags().BoolVar(&flags.optJSON, "json", false, "Output machine-readable JSON summary")
	rootCmd.Flags().BoolVar(&flags.optSummary, "summary", false, "Output single-line text summary")
	rootCmd.Flags().BoolVar(&flags.optHTML, "html", false, "Generate standalone interactive HTML dashboard")
	rootCmd.Flags().BoolVar(&flags.optIframe, "iframe", false, "Generate embeddable minimal HTML (button-free for iframes)")
	rootCmd.Flags().BoolVar(&flags.optIframe, "embed", false, "Alias for --iframe")
	_ = rootCmd.Flags().MarkHidden("embed")

	rootCmd.Flags().StringVarP(&flags.scheme, "scheme", "s", "dark", "Built-in color scheme (dark, light, earth, moss, linen)")
	rootCmd.Flags().StringVarP(&flags.target, "target", "t", "", "Target subnet CIDR (e.g. 192.168.1.0/24) or base IP (auto-detects local subnet if omitted)")
	rootCmd.Flags().IntVarP(&flags.rows, "rows", "r", grid.DefaultRows, "Number of grid rows")
	rootCmd.Flags().IntVarP(&flags.cols, "cols", "c", grid.DefaultCols, "Number of grid columns")
	rootCmd.Flags().IntVarP(&flags.width, "width", "W", grid.DefaultWidth, "Output image width in pixels")
	rootCmd.Flags().IntVarP(&flags.height, "height", "H", grid.DefaultHeight, "Output image height in pixels")
	rootCmd.Flags().IntVar(&flags.borderWidth, "border-width", grid.DefaultBorderWidth, "Grid divider border width in pixels")
	rootCmd.Flags().StringVarP(&flags.outputPath, "output-file", "f", "", "Path for output file (e.g. grid.html, embed.html, or .png)")

	rootCmd.Flags().StringVar(&flags.colorOffline, "color-offline", grid.HexString(grid.DefaultColorOffline), "Hex color for offline/unresponsive hosts")
	rootCmd.Flags().StringVar(&flags.colorOnline, "color-online", grid.HexString(grid.DefaultColorOnline), "Hex color for active/online hosts")
	rootCmd.Flags().StringVar(&flags.colorHighlight, "color-highlight", grid.HexString(grid.DefaultColorHighlight), "Hex color for fast/highlight hosts")
	rootCmd.Flags().StringVar(&flags.colorSlow, "color-slow", grid.HexString(grid.DefaultColorSlow), "Hex color for slow-responding hosts")
	rootCmd.Flags().StringVar(&flags.colorBorder, "color-border", grid.HexString(grid.DefaultColorBorder), "Hex color for cell divider lines")
	rootCmd.Flags().StringVar(&flags.colorFrame, "color-frame", grid.HexString(grid.DefaultColorFrame), "Hex color for outer canvas frame")

	rootCmd.Flags().IntVar(&flags.concurrency, "concurrency", 128, "Number of concurrent ping workers")
	rootCmd.Flags().DurationVar(&flags.fastThreshold, "fast-threshold", 20*time.Millisecond, "Latency threshold for highlight color")
	rootCmd.Flags().DurationVar(&flags.slowThreshold, "slow-threshold", 100*time.Millisecond, "Latency threshold for slow/degraded color")
	rootCmd.Flags().StringVar(&flags.gateway, "gateway", "", "Optional gateway/infrastructure IP to always highlight")
	rootCmd.Flags().DurationVarP(&flags.refresh, "refresh", "R", 0, "Continuous sweep refresh interval (e.g. 5s, 10s; 0 runs once)")
	rootCmd.Flags().BoolVarP(&flags.showVersion, "version", "v", false, "Display version information and exit")

	// Shell tab-completion command
	completionCmd := &cobra.Command{
		Use:   "completion [command]",
		Short: "Generate shell completion script",
		Long: `Generate shell autocompletion script for pg.
Supports tab completion for command flags, built-in color schemes, and output formats.

Usage:
  pg completion powershell
    Outputs a PowerShell script defining Register-ArgumentCompleter.

Examples:
  # Enable completion in current PowerShell session:
  pg completion powershell | Out-String | Invoke-Expression

  # Persist completion in your PowerShell profile:
  Add-Content $PROFILE "` + "`" + `\npg completion powershell | Out-String | Invoke-Expression` + "`" + `"`,
		Hidden: true,
	}
	completionPSCmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell tab completion script",
		Long: `Generate PowerShell tab completion script for pg.
Completes flags (-s/--scheme, -o/--output, -R/--refresh, etc.), valid scheme names
(dark, light, earth, moss, linen), and output formats (text, json, summary, html, iframe).

To load in current session:
  pg completion powershell | Out-String | Invoke-Expression`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.GeneratePowerShellCompletion(rootCmd)
		},
	}
	completionCmd.AddCommand(completionPSCmd)
	rootCmd.AddCommand(completionCmd)

	return rootCmd, &flags
}

type sweepSummary struct {
	Target       string        `json:"target"`
	TotalHosts   int           `json:"total_hosts"`
	OnlineHosts  int           `json:"online_hosts"`
	FastHosts    int           `json:"fast_hosts"`
	SlowHosts    int           `json:"slow_hosts"`
	OfflineHosts int           `json:"offline_hosts"`
	Duration     time.Duration `json:"duration_ms"`
	OutputFile   string        `json:"output_file,omitempty"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Rows         int           `json:"rows"`
	Cols         int           `json:"cols"`
	JoinedHosts  []string      `json:"joined_hosts,omitempty"`
	DroppedHosts []string      `json:"dropped_hosts,omitempty"`
}

func runSweep(cmd *cobra.Command, flags *appFlags, args []string) error {
	// Positional target argument support (e.g. pg 10.8.0.1/24)
	if len(args) > 0 {
		flags.target = args[0]
	}

	cli.ApplyPSMode(&flags.StandardFlags)
	logger := log.New(flags.Quiet)

	// Load base color scheme
	gridCfg, err := grid.GetScheme(flags.scheme)
	if err != nil {
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid color scheme", flags.scheme, "Choose from: dark, light, earth, moss, linen", err)
	}

	// Apply explicit flag overrides if specified
	if cmd.Flags().Changed("color-offline") {
		c, err := grid.ParseHexColor(flags.colorOffline)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-offline", flags.colorOffline, "Provide valid hex color like #404e41", err)
		}
		gridCfg.ColorOffline = c
	}
	if cmd.Flags().Changed("color-online") {
		c, err := grid.ParseHexColor(flags.colorOnline)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-online", flags.colorOnline, "Provide valid hex color like #4d86a2", err)
		}
		gridCfg.ColorOnline = c
	}
	if cmd.Flags().Changed("color-highlight") {
		c, err := grid.ParseHexColor(flags.colorHighlight)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-highlight", flags.colorHighlight, "Provide valid hex color like #e4f9d4", err)
		}
		gridCfg.ColorHighlight = c
	}
	if cmd.Flags().Changed("color-slow") {
		c, err := grid.ParseHexColor(flags.colorSlow)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-slow", flags.colorSlow, "Provide valid hex color like #ab7550", err)
		}
		gridCfg.ColorSlow = c
	}
	if cmd.Flags().Changed("color-border") {
		c, err := grid.ParseHexColor(flags.colorBorder)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-border", flags.colorBorder, "Provide valid hex color like #2c2c2c", err)
		}
		gridCfg.ColorBorder = c
	}
	if cmd.Flags().Changed("color-frame") {
		c, err := grid.ParseHexColor(flags.colorFrame)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-frame", flags.colorFrame, "Provide valid hex color like #2c2c2c", err)
		}
		gridCfg.ColorFrame = c
	}

	if flags.rows <= 0 || flags.cols <= 0 {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Grid rows and columns must be greater than zero", fmt.Sprintf("%dx%d", flags.rows, flags.cols), "Specify positive integers for --rows and --cols", nil)
	}

	if flags.outputPath != "" {
		normPath, err := paths.Normalize(flags.outputPath)
		if err == nil {
			flags.outputPath = normPath
		}
	}

	var formatFlags []string
	if flags.optText {
		formatFlags = append(formatFlags, "--text")
	}
	if flags.optJSON {
		formatFlags = append(formatFlags, "--json")
	}
	if flags.optSummary {
		formatFlags = append(formatFlags, "--summary")
	}
	if flags.optHTML {
		formatFlags = append(formatFlags, "--html")
	}
	if flags.optIframe {
		formatFlags = append(formatFlags, "--iframe")
	}

	if len(formatFlags) > 1 {
		return errors.New(errors.ExitInput, "USAGE_CONFLICTING_FLAGS", "Multiple conflicting output format flags specified", strings.Join(formatFlags, ", "), "Specify only one output format flag (e.g. --html or --json)", nil)
	}

	if len(formatFlags) == 1 {
		switch {
		case flags.optText:
			flags.OutputFormat = "text"
		case flags.optJSON:
			flags.OutputFormat = "json"
		case flags.optSummary:
			flags.OutputFormat = "summary"
		case flags.optHTML:
			flags.OutputFormat = "html"
		case flags.optIframe:
			flags.OutputFormat = "iframe"
		}
	} else if !cmd.Flags().Changed("output") {
		if flags.PSMode {
			flags.OutputFormat = "json"
		} else {
			flags.OutputFormat = "text"
		}
	}

	outFormat := strings.ToLower(strings.TrimSpace(flags.OutputFormat))
	switch outFormat {
	case "text", "console", "":
		flags.OutputFormat = "text"
	case "json", "ndjson":
		flags.OutputFormat = "json"
	case "summary":
		flags.OutputFormat = "summary"
	case "html", "standalone", "html-standalone":
		flags.OutputFormat = "html"
	case "iframe", "embed", "html-embed", "html-iframe":
		flags.OutputFormat = "iframe"
	default:
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid output format", flags.OutputFormat, "Choose from: --text, --json, --summary, --html, --iframe", nil)
	}

	// Default output file path when HTML or iframe is explicitly requested without -f
	if flags.outputPath == "" {
		if flags.OutputFormat == "html" {
			flags.outputPath = "grid.html"
		} else if flags.OutputFormat == "iframe" {
			flags.outputPath = "grid-embed.html"
		}
	}

	gridCfg.Rows = flags.rows
	gridCfg.Cols = flags.cols
	gridCfg.Width = flags.width
	gridCfg.Height = flags.height
	gridCfg.BorderWidth = flags.borderWidth

	totalSlots := flags.rows * flags.cols
	ips, autoGateway, err := scanner.GenerateIPs(flags.target, totalSlots)
	if err != nil {
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid target address or CIDR subnet", flags.target, "Provide a valid IPv4 CIDR (e.g. 192.168.1.0/24) or base IP", err)
	}

	var gwIP net.IP
	if flags.gateway != "" {
		gwIP = net.ParseIP(flags.gateway)
	} else if autoGateway != nil {
		gwIP = autoGateway
	}

	pingTimeout := 400 * time.Millisecond
	if cmd.Flags().Changed("timeout") {
		pingTimeout = flags.Timeout
	}

	scanCfg := scanner.Config{
		Count:         totalSlots,
		Concurrency:   flags.concurrency,
		Timeout:       pingTimeout,
		FastThreshold: flags.fastThreshold,
		SlowThreshold: flags.slowThreshold,
		GatewayIP:     gwIP,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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
		if flags.Verbose && res.Status != scanner.StatusOffline {
			_ = logger.Diagnostic("Host %s responded in %v (%s)", res.IP, res.RTT, res.Status)
		}
	})
	duration := time.Since(start)

	var onlineCount, highlightCount, slowCount, offlineCount int
	for _, r := range results {
		switch r.Status {
		case scanner.StatusOnline:
			onlineCount++
		case scanner.StatusHighlight:
			highlightCount++
		case scanner.StatusSlow:
			slowCount++
		default:
			offlineCount++
		}
	}

	activeCount := onlineCount + highlightCount + slowCount
	deltas := scanner.ComputeDeltas(prevResults, results)

	var joinedHosts, droppedHosts []string
	for _, d := range deltas {
		if d.Kind == scanner.DeltaJoined {
			joinedHosts = append(joinedHosts, d.IP.String())
		} else if d.Kind == scanner.DeltaDropped {
			droppedHosts = append(droppedHosts, d.IP.String())
		}
	}

	_ = logger.Diagnostic("Sweep completed in %v. Active: %d (Fast: %d, Normal: %d, Slow: %d), Offline: %d, Deltas: %d",
		duration, activeCount, highlightCount, onlineCount, slowCount, offlineCount, len(deltas))

	// Save to file if output path is configured
	if flags.outputPath != "" {
		if strings.HasSuffix(strings.ToLower(flags.outputPath), ".png") {
			img := grid.Render(gridCfg, results)
			if err := grid.SavePNG(img, flags.outputPath); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save PNG image", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			_ = logger.Diagnostic("Grid saved to %s (%dx%d, %d rows x %d cols)",
				flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
		} else {
			refreshSec := int(flags.refresh.Seconds())

			if flags.OutputFormat == "iframe" {
				minBytes, err := grid.RenderMinimalHTML(gridCfg, results, duration, refreshSec, deltas)
				if err != nil {
					return nil, errors.New(errors.ExitProcessing, "RENDER_FAILED", "Failed to generate minimal embed HTML", "", "Check grid configuration parameters", err)
				}
				if err := grid.SaveHTML(minBytes, flags.outputPath); err != nil {
					return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save minimal embed HTML file", flags.outputPath, "Ensure target directory exists and is writable", err)
				}
				_ = logger.Diagnostic("Embeddable iframe HTML saved to %s (%dx%d, %d rows x %d cols)",
					flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
			} else {
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
	}

	// Console Output Rendering
	isPlain := flags.Plain || flags.Color == "never"

	switch flags.OutputFormat {
	case "json":
		summary := sweepSummary{
			Target:       flags.target,
			TotalHosts:   len(ips),
			OnlineHosts:  activeCount,
			FastHosts:    highlightCount,
			SlowHosts:    slowCount,
			OfflineHosts: offlineCount,
			Duration:     duration,
			OutputFile:   flags.outputPath,
			Width:        flags.width,
			Height:       flags.height,
			Rows:         flags.rows,
			Cols:         flags.cols,
			JoinedHosts:  joinedHosts,
			DroppedHosts: droppedHosts,
		}
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return nil, err
		}
		_ = logger.Data("%s", string(data))

	case "summary":
		if !flags.Quiet {
			deltaSummary := ""
			if len(deltas) > 0 {
				deltaSummary = fmt.Sprintf(" (+%d joined, -%d dropped)", len(joinedHosts), len(droppedHosts))
			}
			_ = logger.Data("pg sweep completed: %d/%d hosts active%s. Output: %s",
				activeCount, len(ips), deltaSummary, flags.outputPath)
		}

	case "html":
		if !flags.Quiet {
			if flags.outputPath != "" {
				_ = logger.Data("pg: %d/%d hosts active. Standalone HTML dashboard saved to %s", activeCount, len(ips), flags.outputPath)
			}
		}

	case "iframe":
		if !flags.Quiet {
			if flags.outputPath != "" {
				_ = logger.Data("pg: %d/%d hosts active. Embeddable iframe HTML saved to %s", activeCount, len(ips), flags.outputPath)
			}
		}

	default: // "text"
		if !flags.Quiet {
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
			asciiGrid := grid.RenderASCIIDelta(gridCfg, results, deltas, isPlain)
			fmt.Print(asciiGrid)
			if flags.outputPath != "" {
				_ = logger.Data("pg: %d/%d hosts active. Visual grid saved to %s", activeCount, len(ips), flags.outputPath)
			} else {
				_ = logger.Data("pg: %d/%d hosts active.", activeCount, len(ips))
			}
		}
	}

	return results, nil
}
