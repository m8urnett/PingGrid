package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/m8urnett/toolkit/errors"
	"github.com/m8urnett/toolkit/paths"
	"github.com/spf13/cobra"
)

func writeOutputFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return paths.SafeWrite(path, data, true)
}

func plainConsoleOutput(flags *appFlags) bool {
	if flags.Plain || flags.noColor || flags.Color == "never" || os.Getenv("NO_COLOR") != "" {
		return true
	}
	if flags.Color == "always" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err != nil || info.Mode()&os.ModeCharDevice == 0
}

func resolveOutputOptions(cmd *cobra.Command, flags *appFlags) error {
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
			flags.outputFormat = "list"
			if flags.optList != stdoutSentinel && flags.optList != "" {
				flags.outputPath = flags.optList
			}
		case cmd.Flags().Changed("ascii") || cmd.Flags().Changed("text"):
			flags.outputFormat = "ascii"
			if flags.optASCII != stdoutSentinel && flags.optASCII != "" {
				flags.outputPath = flags.optASCII
			}
		case cmd.Flags().Changed("json"):
			flags.outputFormat = "json"
			if flags.optJSON != stdoutSentinel && flags.optJSON != "" {
				flags.outputPath = flags.optJSON
			}
		case cmd.Flags().Changed("summary"):
			flags.outputFormat = "summary"
			if flags.optSummary != stdoutSentinel && flags.optSummary != "" {
				flags.outputPath = flags.optSummary
			}
		case cmd.Flags().Changed("html"):
			flags.outputFormat = "html"
			flags.outputPath = flags.optHTML
		case cmd.Flags().Changed("iframe") || cmd.Flags().Changed("embed"):
			flags.outputFormat = "iframe"
			flags.outputPath = flags.optIframe
		case cmd.Flags().Changed("png"):
			flags.outputFormat = "png"
			flags.outputPath = flags.optPNG
		}
	} else {
		flags.outputFormat = "ascii"
	}

	outFormat := strings.ToLower(strings.TrimSpace(flags.outputFormat))
	switch outFormat {
	case "list":
		flags.outputFormat = "list"
	case "ascii", "text", "console", "":
		flags.outputFormat = "ascii"
	case "json", "ndjson":
		flags.outputFormat = "json"
	case "summary":
		flags.outputFormat = "summary"
	case "html", "standalone", "html-standalone":
		flags.outputFormat = "html"
	case "iframe", "embed", "html-embed", "html-iframe":
		flags.outputFormat = "iframe"
	case "png":
		flags.outputFormat = "png"
	default:
		return errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid output format", flags.outputFormat, "Choose from: -l/--list, --ascii, --json, --summary, --html, --iframe, --png", nil)
	}

	if flags.outputPath != "" {
		normPath, err := paths.Normalize(flags.outputPath)
		if err != nil {
			return errors.New(errors.ExitInput, "PATH_INVALID", "Invalid output path", flags.outputPath, "Provide a writable file path", err)
		}
		flags.outputPath = normPath
	}

	return nil
}
