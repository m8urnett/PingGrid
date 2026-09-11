package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/m8urnett/PingGrid/internal/grid"
	versioninfo "github.com/m8urnett/PingGrid/internal/version"
	"github.com/m8urnett/toolkit/cli"
	"github.com/m8urnett/toolkit/errors"
	"github.com/spf13/cobra"
)

func newRootCmdForVersion(releaseVersion string) (*cobra.Command, *appFlags) {
	var flags appFlags

	rootCmd := &cobra.Command{
		Use:   "pg [target]",
		Short: "Ping sweep subnet visualizer and grid image generator",
		Long: `pg (PingGrid) performs a concurrent ping sweep across a target subnet or IP range
and renders an activity grid (terminal ASCII display, interactive HTML, or PNG image).`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flags.noColor || os.Getenv("NO_COLOR") != "" {
				flags.Color = "never"
			}
			switch strings.ToLower(strings.TrimSpace(flags.Color)) {
			case "auto", "always", "never":
				flags.Color = strings.ToLower(strings.TrimSpace(flags.Color))
				return nil
			default:
				return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid color mode", flags.Color, "Choose from: auto, always, never", nil)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.showVersion {
				fmt.Println(versioninfo.Full(releaseVersion))
				return nil
			}
			if flags.showExamples {
				fmt.Print(buildExamplesText())
				return nil
			}
			if flags.listInterfaces {
				return runListInterfaces(cmd, &flags)
			}
			if flags.optimizeOS {
				if len(args) != 0 {
					return errors.New(errors.ExitUsage, "USAGE_INVALID_ARGUMENT", "OS optimization does not accept a target", strings.Join(args, " "), "Run 'pg optimize-os' or 'pg --optimize-os' without a target", nil)
				}
				return runOptimizeOS(cmd, &flags)
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

	rootCmd.PersistentFlags().StringVar(&flags.Color, "color", "auto", "Set color mode: auto, always, never")
	rootCmd.PersistentFlags().BoolVar(&flags.noColor, "no-color", false, "Disable ANSI color output")
	rootCmd.PersistentFlags().BoolVar(&flags.Plain, "plain", false, "Disable formatting and styling")
	rootCmd.PersistentFlags().BoolVar(&flags.Quiet, "quiet", false, "Suppress nonessential progress")
	rootCmd.PersistentFlags().BoolVar(&flags.Verbose, "verbose", false, "Include additional diagnostic evidence")
	rootCmd.PersistentFlags().DurationVar(&flags.Timeout, "timeout", 5*time.Second, "Set the timeout for each operation")
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
	rootCmd.Flags().StringVarP(&flags.ifaceTarget, "interface", "i", "", "Target network interface by name or index (e.g. 'Wi-Fi', 'eth0', 1)")
	rootCmd.Flags().BoolVarP(&flags.listInterfaces, "interfaces", "I", false, "List detected network interfaces and exit")
	rootCmd.Flags().BoolVarP(&flags.allInterfaces, "all-interfaces", "A", false, "Sweep all active local network interfaces simultaneously")
	rootCmd.Flags().BoolVar(&flags.allInterfaces, "all-ifaces", false, "Alias for --all-interfaces")
	_ = rootCmd.Flags().MarkHidden("all-ifaces")
	rootCmd.Flags().BoolVar(&flags.allInterfaces, "all", false, "Alias for --all-interfaces")
	_ = rootCmd.Flags().MarkHidden("all")
	rootCmd.Flags().IntVarP(&flags.pings, "pings", "p", 3, "Number of ping attempts per host (default 3)")
	rootCmd.Flags().DurationVarP(&flags.refresh, "refresh", "R", 0, "Continuous sweep refresh interval (e.g. 5s, 10s; 0 runs once)")
	rootCmd.Flags().IntVar(&flags.concurrency, "concurrency", 256, "Number of concurrent ping workers")
	rootCmd.Flags().DurationVar(&flags.slowThreshold, "slow-threshold", 100*time.Millisecond, "Latency threshold for slow/degraded color")
	rootCmd.Flags().BoolVar(&flags.showHUD, "hud", true, "Display Interface & Link Health HUD context banner above the grid")
	rootCmd.Flags().BoolVar(&flags.enableARP, "arp", true, "Pre-warm discovery cache from OS neighbor table and detect silent hosts")

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
	rootCmd.Flags().BoolVar(&flags.optimizeOS, "optimize-os", false, "Tune host OS network parameters for fast ping sweeping (requires admin/root)")
	rootCmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Inspect proposed OS optimizations without applying changes")

	// Dedicated optimize-os subcommand
	optCmd := &cobra.Command{
		Use:   "optimize-os",
		Short: "Tune host OS network parameters for fast ping sweeping (requires admin/root)",
		Long: `Inspect and apply kernel network stack optimizations to accelerate ICMP ping sweeping
and neighbor table resolution on the host operating system.

Requires Administrator privileges on Windows and root (sudo) privileges on Linux/macOS.
Use --dry-run to inspect proposed changes without modifying system settings.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptimizeOS(cmd, &flags)
		},
	}
	optCmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Inspect proposed OS optimizations without applying changes")
	rootCmd.AddCommand(optCmd)

	// Shell tab-completion command
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell autocompletion script for pg.
Supports tab completion for command flags, built-in color schemes, and output formats.

Available Shells:
  bash        Generate Bash completion script
  zsh         Generate Zsh completion script
  fish        Generate Fish completion script
  powershell  Generate PowerShell completion script

Examples:
  # Bash (load in current session):
  source <(pg completion bash)

  # Zsh (load in current session):
  source <(pg completion zsh)

  # PowerShell (load in current session):
  pg completion powershell | Out-String | Invoke-Expression`,
		Hidden: true,
	}

	completionBashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate Bash tab completion script",
		Long: `Generate Bash tab completion script for pg.

To load in current session:
  source <(pg completion bash)

To load completions for every new session:
  # Linux:
  pg completion bash > /etc/bash_completion.d/pg
  # macOS (with Homebrew bash-completion):
  pg completion bash > $(brew --prefix)/etc/bash_completion.d/pg`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenBashCompletion(os.Stdout)
		},
	}

	completionZshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate Zsh tab completion script",
		Long: `Generate Zsh tab completion script for pg.

To load in current session:
  source <(pg completion zsh)

To load completions for every new session:
  pg completion zsh > "${fpath[1]}/_pg"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenZshCompletion(os.Stdout)
		},
	}

	completionFishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate Fish tab completion script",
		Long: `Generate Fish tab completion script for pg.

To load in current session:
  pg completion fish | source

To load completions for every new session:
  pg completion fish > ~/.config/fish/completions/pg.fish`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	}

	completionPSCmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell tab completion script",
		Long: `Generate PowerShell tab completion script for pg.
Completes flags (-s/--scheme, -p/--pings, -l/--list, -R/--refresh, etc.), valid scheme names
(dark, light, earth, moss, linen), and output formats (list, ascii, json, summary, html, iframe, png).

To load in current session:
  pg completion powershell | Out-String | Invoke-Expression

To persist across all PowerShell sessions:
  Add-Content $PROFILE "` + "`" + `\npg completion powershell | Out-String | Invoke-Expression` + "`" + `"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.GeneratePowerShellCompletion(rootCmd)
		},
	}

	completionCmd.AddCommand(completionBashCmd, completionZshCmd, completionFishCmd, completionPSCmd)
	rootCmd.AddCommand(completionCmd)

	_ = rootCmd.RegisterFlagCompletionFunc("scheme", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return grid.AvailableSchemes(), cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("color", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp
	})
	return rootCmd, &flags
}
