package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
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

const stdoutSentinel = ":stdout"

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
	slowThreshold  time.Duration
	refresh        time.Duration
	showVersion    bool
	showExamples   bool
	pings          int

	// Dedicated output format flags
	optASCII   string
	optJSON    string
	optSummary string
	optHTML    string
	optIframe  string
	optPNG     string
	optList    string
}

func looksLikeTarget(s string) bool {
	if _, _, err := net.ParseCIDR(s); err == nil {
		return true
	}
	if net.ParseIP(s) != nil {
		return true
	}
	if parts := strings.Split(s, "-"); len(parts) == 2 {
		if net.ParseIP(parts[0]) != nil {
			return true
		}
	}
	return false
}

func normalizeFlagToken(token string) (string, bool) {
	if token == "-" || token == "--" {
		return token, false
	}

	var raw string
	if strings.HasPrefix(token, "--") {
		raw = token[2:]
	} else if strings.HasPrefix(token, "-") || strings.HasPrefix(token, "/") {
		raw = token[1:]
	} else {
		return token, false
	}

	if raw == "" {
		return token, false
	}

	// Split name and optional value delimited by = or :
	name := raw
	val := ""
	hasVal := false
	if sep := strings.IndexAny(raw, "=:"); sep != -1 {
		name = raw[:sep]
		val = raw[sep+1:]
		hasVal = true
	}

	// Single-letter shorthands (case-sensitive where collisions exist: R vs r, W/H vs h)
	switch name {
	case "R":
		if hasVal {
			return "-R=" + val, true
		}
		return "-R", true
	case "r":
		if hasVal {
			return "-r=" + val, true
		}
		return "-r", true
	case "W":
		if hasVal {
			return "-W=" + val, true
		}
		return "-W", true
	case "H":
		if hasVal {
			return "-H=" + val, true
		}
		return "-H", true
	case "h", "?":
		return "--help", true
	case "l", "L":
		if hasVal {
			return "-l=" + val, true
		}
		return "-l", true
	case "p", "P":
		if hasVal {
			return "-p=" + val, true
		}
		return "-p", true
	case "c", "C":
		if hasVal {
			return "-c=" + val, true
		}
		return "-c", true
	case "s", "S":
		if hasVal {
			return "-s=" + val, true
		}
		return "-s", true
	case "v", "V":
		return "-v", true
	case "q", "Q":
		return "-q", true
	}

	// Multi-character long flag mapping (case-insensitive)
	nameLower := strings.ToLower(name)
	switch nameLower {
	case "help":
		return "--help", true
	case "version", "ver":
		return "--version", true
	case "examples", "example":
		return "--examples", true
	case "list":
		if hasVal {
			return "--list=" + val, true
		}
		return "--list", true
	case "ascii", "text":
		if hasVal {
			return "--ascii=" + val, true
		}
		return "--ascii", true
	case "json":
		if hasVal {
			return "--json=" + val, true
		}
		return "--json", true
	case "summary":
		if hasVal {
			return "--summary=" + val, true
		}
		return "--summary", true
	case "html":
		if hasVal {
			return "--html=" + val, true
		}
		return "--html", true
	case "iframe", "embed":
		if hasVal {
			return "--iframe=" + val, true
		}
		return "--iframe", true
	case "png":
		if hasVal {
			return "--png=" + val, true
		}
		return "--png", true
	case "pings", "ping":
		if hasVal {
			return "--pings=" + val, true
		}
		return "--pings", true
	case "refresh":
		if hasVal {
			return "--refresh=" + val, true
		}
		return "--refresh", true
	case "concurrency":
		if hasVal {
			return "--concurrency=" + val, true
		}
		return "--concurrency", true
	case "timeout":
		if hasVal {
			return "--timeout=" + val, true
		}
		return "--timeout", true
	case "slow-threshold", "slowthreshold", "slow":
		if hasVal {
			return "--slow-threshold=" + val, true
		}
		return "--slow-threshold", true
	case "rows", "row":
		if hasVal {
			return "--rows=" + val, true
		}
		return "--rows", true
	case "cols", "col", "columns", "column":
		if hasVal {
			return "--cols=" + val, true
		}
		return "--cols", true
	case "width":
		if hasVal {
			return "--width=" + val, true
		}
		return "--width", true
	case "height":
		if hasVal {
			return "--height=" + val, true
		}
		return "--height", true
	case "border-width", "borderwidth":
		if hasVal {
			return "--border-width=" + val, true
		}
		return "--border-width", true
	case "scheme":
		if hasVal {
			return "--scheme=" + val, true
		}
		return "--scheme", true
	case "color-offline", "coloroffline":
		if hasVal {
			return "--color-offline=" + val, true
		}
		return "--color-offline", true
	case "color-online", "coloronline":
		if hasVal {
			return "--color-online=" + val, true
		}
		return "--color-online", true
	case "color-highlight", "colorhighlight":
		if hasVal {
			return "--color-highlight=" + val, true
		}
		return "--color-highlight", true
	case "color-slow", "colorslow":
		if hasVal {
			return "--color-slow=" + val, true
		}
		return "--color-slow", true
	case "color-border", "colorborder":
		if hasVal {
			return "--color-border=" + val, true
		}
		return "--color-border", true
	case "color-frame", "colorframe":
		if hasVal {
			return "--color-frame=" + val, true
		}
		return "--color-frame", true
	case "plain":
		return "--plain", true
	case "verbose":
		return "-v", true
	case "quiet":
		return "-q", true
	case "color":
		if hasVal {
			return "--color=" + val, true
		}
		return "--color", true
	case "fail-on-warning", "failonwarning":
		return "--fail-on-warning", true
	case "encoding":
		if hasVal {
			return "--encoding=" + val, true
		}
		return "--encoding", true
	case "debug-argv", "debugargv":
		return "--debug-argv", true
	case "log":
		if hasVal {
			return "--log=" + val, true
		}
		return "--log", true
	case "no-log", "nolog":
		return "--no-log", true
	case "redact":
		return "--redact", true
	}

	// Fallback for unrecognized switches starting with '/'
	if strings.HasPrefix(token, "/") {
		if len(raw) == 1 {
			return "-" + raw, false
		}
		if hasVal {
			return "--" + name + "=" + val, false
		}
		return "--" + raw, false
	}

	return token, false
}

func normalizeCLIArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	result := make([]string, 0, len(args))
	result = append(result, args[0])

	outputOptionFlags := map[string]bool{
		"--list":    true,
		"-l":        true,
		"--html":    true,
		"--iframe":  true,
		"--embed":   true,
		"--png":     true,
		"--json":    true,
		"--summary": true,
		"--ascii":   true,
		"--text":    true,
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]

		// Support naked keywords
		switch strings.ToLower(arg) {
		case "examples":
			result = append(result, "--examples")
			continue
		case "version", "ver":
			result = append(result, "--version")
			continue
		}

		normArg, _ := normalizeFlagToken(arg)

		if outputOptionFlags[normArg] {
			// Check if next arg is a filename (not another flag and not an IP/CIDR target)
			if i+1 < len(args) {
				next := args[i+1]
				if !strings.HasPrefix(next, "-") && !strings.HasPrefix(next, "/") && !looksLikeTarget(next) {
					result = append(result, normArg+"="+next)
					i++ // Skip consumed filename
					continue
				}
			}
		}

		result = append(result, normArg)
	}

	return result
}

func writeOutputFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0600)
}

func main() {
	os.Args = normalizeCLIArgs(os.Args)

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
			if flags.showExamples {
				fmt.Print(buildExamplesText())
				return nil
			}
			return runSweep(cmd, &flags, args)
		},
	}

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Print(buildHelpText(cmd))
	})
	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Print(buildHelpText(cmd))
		return nil
	})
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	cli.BindStandardFlags(rootCmd, &flags.StandardFlags)
	if f := rootCmd.PersistentFlags().Lookup("quiet"); f != nil {
		f.Shorthand = "q"
	}
	if f := rootCmd.PersistentFlags().Lookup("verbose"); f != nil {
		f.Shorthand = "v"
	}

	// Output Options (accepting optional filenames)
	rootCmd.Flags().StringVarP(&flags.optList, "list", "l", "", "Output sorted list of host names and ping times (optionally write to file)")
	rootCmd.Flags().Lookup("list").NoOptDefVal = stdoutSentinel

	rootCmd.Flags().StringVar(&flags.optASCII, "ascii", "", "Output console ASCII text grid (optionally write to file)")
	rootCmd.Flags().Lookup("ascii").NoOptDefVal = stdoutSentinel

	rootCmd.Flags().StringVar(&flags.optASCII, "text", "", "Alias for --ascii")
	rootCmd.Flags().Lookup("text").NoOptDefVal = stdoutSentinel
	_ = rootCmd.Flags().MarkHidden("text")

	rootCmd.Flags().StringVar(&flags.optJSON, "json", "", "Output machine-readable JSON summary (optionally write to file)")
	rootCmd.Flags().Lookup("json").NoOptDefVal = stdoutSentinel

	rootCmd.Flags().StringVar(&flags.optSummary, "summary", "", "Output single-line text summary (optionally write to file)")
	rootCmd.Flags().Lookup("summary").NoOptDefVal = stdoutSentinel

	rootCmd.Flags().StringVar(&flags.optHTML, "html", "", "Generate standalone interactive HTML dashboard (default: grid.html)")
	rootCmd.Flags().Lookup("html").NoOptDefVal = "grid.html"

	rootCmd.Flags().StringVar(&flags.optIframe, "iframe", "", "Generate embeddable minimal HTML for iframes (default: grid-embed.html)")
	rootCmd.Flags().Lookup("iframe").NoOptDefVal = "grid-embed.html"

	rootCmd.Flags().StringVar(&flags.optIframe, "embed", "", "Alias for --iframe")
	rootCmd.Flags().Lookup("embed").NoOptDefVal = "grid-embed.html"
	_ = rootCmd.Flags().MarkHidden("embed")

	rootCmd.Flags().StringVar(&flags.optPNG, "png", "", "Generate PNG activity grid image (default: grid.png)")
	rootCmd.Flags().Lookup("png").NoOptDefVal = "grid.png"

	// Scan Options
	rootCmd.Flags().IntVarP(&flags.pings, "pings", "p", 3, "Number of ping attempts per host (default 3)")
	rootCmd.Flags().DurationVarP(&flags.refresh, "refresh", "R", 0, "Continuous sweep refresh interval (e.g. 5s, 10s; 0 runs once)")
	rootCmd.Flags().IntVar(&flags.concurrency, "concurrency", 128, "Number of concurrent ping workers")
	rootCmd.Flags().DurationVar(&flags.slowThreshold, "slow-threshold", 100*time.Millisecond, "Latency threshold for slow/degraded color")

	// Grid Layout Options
	rootCmd.Flags().IntVarP(&flags.rows, "rows", "r", grid.DefaultRows, "Number of grid rows (auto-sized to fit IP range if omitted)")
	rootCmd.Flags().IntVarP(&flags.cols, "cols", "c", grid.DefaultCols, "Number of grid columns (auto-sized to fit IP range if omitted)")
	rootCmd.Flags().IntVarP(&flags.width, "width", "W", grid.DefaultWidth, "Output image width in pixels (auto-sized if omitted)")
	rootCmd.Flags().IntVarP(&flags.height, "height", "H", grid.DefaultHeight, "Output image height in pixels (auto-sized if omitted)")
	rootCmd.Flags().IntVar(&flags.borderWidth, "border-width", grid.DefaultBorderWidth, "Grid divider border width in pixels")

	// Color & Styling Options
	rootCmd.Flags().StringVarP(&flags.scheme, "scheme", "s", "dark", "Built-in color scheme (dark, light, earth, moss, linen)")
	rootCmd.Flags().StringVar(&flags.colorOffline, "color-offline", grid.HexString(grid.DefaultColorOffline), "Hex color for offline/unresponsive hosts (with or without #)")
	rootCmd.Flags().StringVar(&flags.colorOnline, "color-online", grid.HexString(grid.DefaultColorOnline), "Hex color for active/online hosts (with or without #)")
	rootCmd.Flags().StringVar(&flags.colorHighlight, "color-highlight", grid.HexString(grid.DefaultColorHighlight), "Hex color for fast/highlight hosts (with or without #)")
	rootCmd.Flags().StringVar(&flags.colorSlow, "color-slow", grid.HexString(grid.DefaultColorSlow), "Hex color for slow-responding hosts (with or without #)")
	rootCmd.Flags().StringVar(&flags.colorBorder, "color-border", grid.HexString(grid.DefaultColorBorder), "Hex color for cell divider lines (with or without #)")
	rootCmd.Flags().StringVar(&flags.colorFrame, "color-frame", grid.HexString(grid.DefaultColorFrame), "Hex color for outer canvas frame (with or without #)")

	// Standard Options
	rootCmd.Flags().BoolVar(&flags.showVersion, "version", false, "Display version information and exit")
	rootCmd.Flags().BoolVar(&flags.showVersion, "ver", false, "Display version information and exit")
	_ = rootCmd.Flags().MarkHidden("ver")
	rootCmd.Flags().BoolVar(&flags.showExamples, "examples", false, "Display usage examples and target range formats")

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
Completes flags (-s/--scheme, -p/--pings, -l/--list, -R/--refresh, etc.), valid scheme names
(dark, light, earth, moss, linen), and output formats (list, text, json, summary, html, iframe, png).

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

func buildHelpText(cmd *cobra.Command) string {
	return `pg (PingGrid) performs a concurrent ping sweep across a target subnet or IP range
and renders an activity grid (terminal ASCII display, interactive HTML, or PNG image).

Usage:
  pg [target] [flags]

Output Options:
  -l, --list [file]              Output sorted list of host names and ping times (optionally write to file)
      --html [file]              Generate standalone interactive HTML dashboard (default: grid.html)
      --iframe [file]            Generate embeddable minimal HTML for iframes (default: grid-embed.html)
      --png [file]               Generate PNG activity grid image (default: grid.png)
      --json [file]              Output machine-readable JSON summary (optionally write to file)
      --summary [file]           Output single-line text summary (optionally write to file)
      --ascii [file]             Output console ASCII text grid (optionally write to file)

Scan Options:
  -p, --pings <count>            Number of ping attempts per host (default 3)
  -R, --refresh <interval>       Continuous sweep refresh interval (e.g. 5s, 10s; 0 runs once)
      --concurrency <workers>    Number of concurrent ping workers (default 128)
      --timeout <duration>       Ping timeout duration per host (default 150ms RFC1918/LAN, 400ms WAN)
      --slow-threshold <duration> Latency threshold for slow/degraded color (default 100ms)

Grid Layout Options:
  -r, --rows <count>             Number of grid rows (auto-sized to fit IP range if omitted)
  -c, --cols <count>             Number of grid columns (auto-sized to fit IP range if omitted)
  -W, --width <pixels>           Output image width in pixels (auto-sized if omitted)
  -H, --height <pixels>          Output image height in pixels (auto-sized if omitted)
      --border-width <pixels>    Grid divider border width in pixels (default 1)

Color & Styling Options:
  -s, --scheme <name>            Built-in color scheme (dark, light, earth, moss, linen) (default "dark")
      --color-offline <color>    Hex color for offline/unresponsive hosts (with or without #)
      --color-online <color>     Hex color for active/online hosts (with or without #)
      --color-highlight <color>  Hex color for fast/highlight hosts (with or without #)
      --color-slow <color>       Hex color for slow-responding hosts (with or without #)
      --color-border <color>     Hex color for cell divider lines (with or without #)
      --color-frame <color>      Hex color for outer canvas frame (with or without #)

Standard Options:
      --version, --ver           Display version information and exit
  -v, --verbose                  Enable detailed diagnostic output
  -q, --quiet                    Suppress non-essential console output
  -h, --help                     Display help and exit
      --examples                 Display detailed usage examples and target formats
      --plain                    Plain monochrome ASCII mode without ANSI colors

Examples:
  Run 'pg --examples' to view detailed usage examples and all supported IP range formats.

Shell Completion:
  Generate tab autocompletion script for commands, flags, and color schemes.

  PowerShell (load in current session):
    pg completion powershell | Out-String | Invoke-Expression

  PowerShell (persist across all sessions in $PROFILE):
    Add-Content $PROFILE "` + "`" + `\npg completion powershell | Out-String | Invoke-Expression` + "`" + `"

  Completes all flags (-s/--scheme, -p/--pings, --html, --png, --json, -R/--refresh, etc.), built-in
  color schemes (dark, light, earth, moss, linen), and options.
`
}

func buildExamplesText() string {
	return `PingGrid (pg) - Usage Examples & Target Range Formats

1. Supported IP Target Range Formats:
  # Single IPv4 address:
  pg 192.168.1.50

  # Standard CIDR subnet prefix:
  pg 192.168.1.0/24
  pg 10.0.0.0/28

  # Shorthand octet range:
  pg 192.168.1.1-50

  # Full IP start-end range:
  pg 192.168.1.100-192.168.1.200

  # Wildcard pattern (expands full octet 0-255):
  pg 192.168.1.*

  # Bracket range pattern across octets:
  pg 192.168.1.[1-30]
  pg 10.0.[1-2].[1-10]

  # Subnet with explicit netmask:
  pg 192.168.1.0/255.255.255.0

  # Comma-separated combination of addresses & ranges:
  pg 192.168.1.1,192.168.1.254,10.0.0.1-10

  # Auto-detected local subnet (omitting target scans active /24):
  pg

2. Output Formats:
  # Sorted list of responding hosts (hostnames & ping times):
  pg 192.168.1.0/24 -l

  # Save sorted host list to file:
  pg 192.168.1.0/24 --list hosts.txt

  # Standalone interactive HTML dashboard (saves to grid.html):
  pg 192.168.1.0/24 --html

  # Standalone HTML dashboard with custom filename:
  pg 192.168.1.0/24 --html dashboard.html

  # Minimal embeddable HTML for iframes:
  pg 192.168.1.0/24 --iframe embed.html

  # PNG activity grid image (default: grid.png):
  pg 192.168.1.0/24 --png

  # PNG activity grid image with custom filename:
  pg 192.168.1.0/24 --png report.png

  # Machine-readable JSON summary (to stdout or file):
  pg 192.168.1.0/24 --json
  pg 192.168.1.0/24 --json scan.json

  # Single-line text summary:
  pg 192.168.1.0/24 --summary

  # Plain monochrome ASCII mode (no ANSI color escapes):
  pg 192.168.1.0/24 --plain

3. Scan Configuration & Tuning:
  # Multi-ping sweep (3 attempts per host, recording lowest RTT):
  pg 192.168.1.0/24 -p 3

  # Continuous live monitoring (refresh sweep every 5 seconds):
  pg 192.168.1.0/24 -R 5s

  # Adjust ping timeout (defaults to 150ms LAN, 400ms WAN):
  pg 10.0.0.0/24 --timeout 250ms

  # Set custom slow-response latency threshold:
  pg 192.168.1.0/24 --slow-threshold 50ms

  # Tune concurrency (worker pool size, default: 128):
  pg 10.0.0.0/16 --concurrency 256

4. Color Themes & Palette Customization:
  # Switch built-in scheme (dark, light, earth, moss, linen):
  pg 192.168.1.0/24 -s light
  pg 192.168.1.0/24 -s moss

  # Custom hex colors (with or without leading #):
  pg 192.168.1.0/24 --color-online 4d86a2 --color-offline 404e41
  pg 192.168.1.0/24 --color-online "#4d86a2" --color-slow "#ab7550"
`
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
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-offline", flags.colorOffline, "Provide valid hex color like #404e41 or 404e41", err)
		}
		gridCfg.ColorOffline = c
	}
	if cmd.Flags().Changed("color-online") {
		c, err := grid.ParseHexColor(flags.colorOnline)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-online", flags.colorOnline, "Provide valid hex color like #4d86a2 or 4d86a2", err)
		}
		gridCfg.ColorOnline = c
	}
	if cmd.Flags().Changed("color-highlight") {
		c, err := grid.ParseHexColor(flags.colorHighlight)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-highlight", flags.colorHighlight, "Provide valid hex color like #e4f9d4 or e4f9d4", err)
		}
		gridCfg.ColorHighlight = c
	}
	if cmd.Flags().Changed("color-slow") {
		c, err := grid.ParseHexColor(flags.colorSlow)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-slow", flags.colorSlow, "Provide valid hex color like #ab7550 or ab7550", err)
		}
		gridCfg.ColorSlow = c
	}
	if cmd.Flags().Changed("color-border") {
		c, err := grid.ParseHexColor(flags.colorBorder)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-border", flags.colorBorder, "Provide valid hex color like #2c2c2c or 2c2c2c", err)
		}
		gridCfg.ColorBorder = c
	}
	if cmd.Flags().Changed("color-frame") {
		c, err := grid.ParseHexColor(flags.colorFrame)
		if err != nil {
			return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid --color-frame", flags.colorFrame, "Provide valid hex color like #2c2c2c or 2c2c2c", err)
		}
		gridCfg.ColorFrame = c
	}

	if cmd.Flags().Changed("rows") && flags.rows <= 0 {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Grid rows must be greater than zero", fmt.Sprintf("%d", flags.rows), "Specify a positive integer for --rows", nil)
	}
	if cmd.Flags().Changed("cols") && flags.cols <= 0 {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Grid columns must be greater than zero", fmt.Sprintf("%d", flags.cols), "Specify a positive integer for --cols", nil)
	}

	var formatFlags []string
	if cmd.Flags().Changed("list") {
		formatFlags = append(formatFlags, "--list")
	}
	if cmd.Flags().Changed("ascii") || cmd.Flags().Changed("text") {
		formatFlags = append(formatFlags, "--ascii")
	}
	if cmd.Flags().Changed("json") {
		formatFlags = append(formatFlags, "--json")
	}
	if cmd.Flags().Changed("summary") {
		formatFlags = append(formatFlags, "--summary")
	}
	if cmd.Flags().Changed("html") {
		formatFlags = append(formatFlags, "--html")
	}
	if cmd.Flags().Changed("iframe") || cmd.Flags().Changed("embed") {
		formatFlags = append(formatFlags, "--iframe")
	}
	if cmd.Flags().Changed("png") {
		formatFlags = append(formatFlags, "--png")
	}

	if len(formatFlags) > 1 {
		return errors.New(errors.ExitInput, "USAGE_CONFLICTING_FLAGS", "Multiple conflicting output format flags specified", strings.Join(formatFlags, ", "), "Specify only one output format flag (e.g. --list, --html, or --json)", nil)
	}

	if len(formatFlags) == 1 {
		switch {
		case cmd.Flags().Changed("list"):
			flags.OutputFormat = "list"
			if flags.optList != stdoutSentinel && flags.optList != "" {
				flags.outputPath = flags.optList
			}
		case cmd.Flags().Changed("ascii") || cmd.Flags().Changed("text"):
			flags.OutputFormat = "ascii"
			if flags.optASCII != stdoutSentinel && flags.optASCII != "" {
				flags.outputPath = flags.optASCII
			}
		case cmd.Flags().Changed("json"):
			flags.OutputFormat = "json"
			if flags.optJSON != stdoutSentinel && flags.optJSON != "" {
				flags.outputPath = flags.optJSON
			}
		case cmd.Flags().Changed("summary"):
			flags.OutputFormat = "summary"
			if flags.optSummary != stdoutSentinel && flags.optSummary != "" {
				flags.outputPath = flags.optSummary
			}
		case cmd.Flags().Changed("html"):
			flags.OutputFormat = "html"
			flags.outputPath = flags.optHTML
		case cmd.Flags().Changed("iframe") || cmd.Flags().Changed("embed"):
			flags.OutputFormat = "iframe"
			flags.outputPath = flags.optIframe
		case cmd.Flags().Changed("png"):
			flags.OutputFormat = "png"
			flags.outputPath = flags.optPNG
		}
	} else {
		flags.OutputFormat = "ascii"
	}

	outFormat := strings.ToLower(strings.TrimSpace(flags.OutputFormat))
	switch outFormat {
	case "list":
		flags.OutputFormat = "list"
	case "ascii", "text", "console", "":
		flags.OutputFormat = "ascii"
	case "json", "ndjson":
		flags.OutputFormat = "json"
	case "summary":
		flags.OutputFormat = "summary"
	case "html", "standalone", "html-standalone":
		flags.OutputFormat = "html"
	case "iframe", "embed", "html-embed", "html-iframe":
		flags.OutputFormat = "iframe"
	case "png":
		flags.OutputFormat = "png"
	default:
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid output format", flags.OutputFormat, "Choose from: -l/--list, --ascii, --json, --summary, --html, --iframe, --png", nil)
	}

	if flags.outputPath != "" {
		normPath, err := paths.Normalize(flags.outputPath)
		if err == nil {
			flags.outputPath = normPath
		}
	}

	if flags.pings <= 0 {
		return errors.New(errors.ExitInput, "INPUT_OUT_OF_RANGE", "Number of pings must be greater than zero", fmt.Sprintf("%d", flags.pings), "Specify a positive integer for -p/--pings (default 1)", nil)
	}

	ips, autoGateway, err := scanner.GenerateIPs(flags.target, 0)
	if err != nil {
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid target address or CIDR subnet", flags.target, "Provide a valid IPv4 CIDR (e.g. 192.168.1.0/24) or base IP", err)
	}

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

	gridCfg.Rows = flags.rows
	gridCfg.Cols = flags.cols
	gridCfg.Width = flags.width
	gridCfg.Height = flags.height
	gridCfg.BorderWidth = flags.borderWidth

	var gwIP net.IP
	if autoGateway != nil {
		gwIP = autoGateway
	}

	pingTimeout := scanner.ResolveDefaultTimeout(ips)
	if cmd.Flags().Changed("timeout") {
		pingTimeout = flags.Timeout
	}

	scanCfg := scanner.Config{
		Count:         len(ips),
		Concurrency:   flags.concurrency,
		Timeout:       pingTimeout,
		SlowThreshold: flags.slowThreshold,
		GatewayIP:     gwIP,
		Pings:         flags.pings,
		BroadcastIPs:  scanner.ExtractBroadcastIPs(flags.target),
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
			_ = logger.Diagnostic("Host %s responded in %s (%s)", res.IP, scanner.FormatDurationMS(res.RTT), res.Status)
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
		switch d.Kind {
		case scanner.DeltaJoined:
			joinedHosts = append(joinedHosts, d.IP.String())
		case scanner.DeltaDropped:
			droppedHosts = append(droppedHosts, d.IP.String())
		}
	}

	_ = logger.Diagnostic("Sweep completed in %s. Active: %d (Fast: %d, Normal: %d, Slow: %d), Offline: %d, Deltas: %d",
		scanner.FormatDurationMS(duration), activeCount, highlightCount, onlineCount, slowCount, offlineCount, len(deltas))

	// Save to file if output path is configured
	if flags.outputPath != "" {
		if flags.OutputFormat == "png" || strings.HasSuffix(strings.ToLower(flags.outputPath), ".png") {
			img := grid.Render(gridCfg, results)
			if err := grid.SavePNG(img, flags.outputPath); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save PNG image", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			_ = logger.Diagnostic("Grid saved to %s (%dx%d, %d rows x %d cols)",
				flags.outputPath, flags.width, flags.height, flags.rows, flags.cols)
		} else if flags.OutputFormat == "iframe" {
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
		} else if flags.OutputFormat == "html" {
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
	isPlain := flags.Plain || flags.Color == "never"

	switch flags.OutputFormat {
	case "png":
		if !flags.Quiet {
			_ = logger.Data("pg: %d/%d hosts active. Activity grid image saved to %s", activeCount, len(ips), flags.outputPath)
		}

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
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, data); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save JSON summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Data("pg: %d/%d hosts active. JSON summary saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else {
			_ = logger.Data("%s", string(data))
		}

	case "summary":
		deltaSummary := ""
		if len(deltas) > 0 {
			deltaSummary = fmt.Sprintf(" (+%d joined, -%d dropped)", len(joinedHosts), len(droppedHosts))
		}
		summaryLine := fmt.Sprintf("%d/%d hosts active%s. Output: %s",
			activeCount, len(ips), deltaSummary, flags.outputPath)
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, []byte(summaryLine+"\n")); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save summary file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Data("pg: %d/%d hosts active. Summary saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else if !flags.Quiet {
			_ = logger.Data("%s", summaryLine)
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

	case "list":
		resolved := scanner.ResolveHostnames(ctx, results, 200*time.Millisecond)
		if flags.outputPath != "" {
			listText := grid.RenderList(resolved, true, flags.Verbose, len(ips), deltas)
			if err := writeOutputFile(flags.outputPath, []byte(listText)); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save host list file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Data("pg: %d/%d hosts active. Host list saved to %s", activeCount, len(ips), flags.outputPath)
			}
		} else if !flags.Quiet {
			listText := grid.RenderList(resolved, isPlain, flags.Verbose, len(ips), deltas)
			if iteration > 1 {
				fmt.Printf("\n--- pg Sweep #%d (%s) ---\n", iteration, time.Now().Format("15:04:05"))
			}
			fmt.Print(listText)
		}

	default: // "ascii"
		asciiGrid := grid.RenderASCIIDelta(gridCfg, results, deltas, isPlain)
		if flags.outputPath != "" {
			if err := writeOutputFile(flags.outputPath, []byte(asciiGrid)); err != nil {
				return nil, errors.New(errors.ExitOutput, errors.CodeOutputWriteFailed, "Failed to save ASCII grid file", flags.outputPath, "Ensure target directory exists and is writable", err)
			}
			if !flags.Quiet {
				_ = logger.Data("pg: %d/%d hosts active. Visual grid saved to %s", activeCount, len(ips), flags.outputPath)
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
			_ = logger.Data("pg: %d/%d hosts active.", activeCount, len(ips))
		}
	}

	return results, nil
}
