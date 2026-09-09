package cli

import (
	"time"

	"github.com/spf13/cobra"
)

// StandardFlags represents the common CLI flags required by the coding standards.
type StandardFlags struct {
	PSMode        bool
	FailOnWarning bool
	Color         string
	Plain         bool
	Encoding      string
	DebugArgv     bool
	Yes           bool
	NoInput       bool
	OutputFormat  string
	Quiet         bool
	Verbose       bool
	Timeout       time.Duration
	LogPath       string
	NoLog         bool
	Redact        bool
}

// BindStandardFlags adds the standard flags to the given cobra command.
func BindStandardFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.PersistentFlags().BoolVar(&flags.PSMode, "ps", false, "Enable PowerShell compatibility mode (implies --output json --no-color --no-progress --encoding utf8)")
	cmd.PersistentFlags().BoolVar(&flags.FailOnWarning, "fail-on-warning", false, "Exit with error if warnings occur")
	cmd.PersistentFlags().StringVar(&flags.Color, "color", "auto", "Set color mode: auto, always, never")
	cmd.PersistentFlags().BoolVar(&flags.Plain, "plain", false, "Disable formatting and styling")
	cmd.PersistentFlags().StringVar(&flags.Encoding, "encoding", "utf8", "Set output encoding (e.g. utf8, utf8-bom)")
	cmd.PersistentFlags().BoolVar(&flags.DebugArgv, "debug-argv", false, "Print raw arguments and exit")
	cmd.PersistentFlags().BoolVar(&flags.Yes, "yes", false, "Assume yes for all prompts")
	cmd.PersistentFlags().BoolVar(&flags.NoInput, "no-input", false, "Disable all interactive prompts")
	cmd.PersistentFlags().StringVarP(&flags.OutputFormat, "output", "o", "text", "Set output format (text, json, ndjson, csv, table)")
	cmd.PersistentFlags().BoolVar(&flags.Quiet, "quiet", false, "Suppress nonessential progress")
	cmd.PersistentFlags().BoolVar(&flags.Verbose, "verbose", false, "Include additional diagnostic evidence")
	cmd.PersistentFlags().DurationVar(&flags.Timeout, "timeout", 5*time.Second, "Set the timeout for each operation")
	cmd.PersistentFlags().StringVar(&flags.LogPath, "log", "", "Write a persistent report to this path")
	cmd.PersistentFlags().BoolVar(&flags.NoLog, "no-log", false, "Disable persistent logging")
	cmd.PersistentFlags().BoolVar(&flags.Redact, "redact", false, "Redact sensitive values in reports")
}

// ApplyPSMode overrides standard flags if --ps mode is enabled.
func ApplyPSMode(flags *StandardFlags) {
	if flags.PSMode {
		flags.OutputFormat = "json"
		flags.Color = "never"
		flags.Encoding = "utf8"
		flags.Plain = true
	}
}
