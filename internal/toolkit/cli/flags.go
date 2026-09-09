package cli

import (
	"time"

	"github.com/spf13/cobra"
)

// StandardFlags represents the common CLI flags required by the coding standards.
type StandardFlags struct {
	FailOnWarning bool
	Color         string
	Plain         bool
	Encoding      string
	DebugArgv     bool
	OutputFormat  string
	Quiet         bool
	Verbose       bool
	Timeout       time.Duration
	LogPath       string
	NoLog         bool
}

// BindStandardFlags adds the standard flags to the given cobra command.
func BindStandardFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.PersistentFlags().BoolVar(&flags.FailOnWarning, "fail-on-warning", false, "Exit with error if warnings occur")
	cmd.PersistentFlags().StringVar(&flags.Color, "color", "auto", "Set color mode: auto, always, never")
	cmd.PersistentFlags().BoolVar(&flags.Plain, "plain", false, "Disable formatting and styling")
	cmd.PersistentFlags().StringVar(&flags.Encoding, "encoding", "utf8", "Set output encoding (e.g. utf8, utf8-bom)")
	cmd.PersistentFlags().BoolVar(&flags.DebugArgv, "debug-argv", false, "Print raw arguments and exit")
	cmd.PersistentFlags().BoolVar(&flags.Quiet, "quiet", false, "Suppress nonessential progress")
	cmd.PersistentFlags().BoolVar(&flags.Verbose, "verbose", false, "Include additional diagnostic evidence")
	cmd.PersistentFlags().DurationVar(&flags.Timeout, "timeout", 5*time.Second, "Set the timeout for each operation")
	cmd.PersistentFlags().StringVar(&flags.LogPath, "log", "", "Write a persistent report to this path")
	cmd.PersistentFlags().BoolVar(&flags.NoLog, "no-log", false, "Disable persistent logging")
}
