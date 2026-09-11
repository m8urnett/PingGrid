package app

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m8urnett/PingGrid/internal/grid"
	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/log"
	"github.com/spf13/cobra"
)

func newRootCmd() (*cobra.Command, *appFlags) {
	return newRootCmdForVersion("test")
}

func TestExecuteHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := Execute(ctx, []string{"pg", "127.0.0.1", "--summary", "--quiet"}, "test"); err != nil {
		t.Fatalf("Execute returned an error for canceled context: %v", err)
	}
}

func TestScanFlagUpperBounds(t *testing.T) {
	tests := [][]string{
		{"127.0.0.1", "--pings", "11", "--summary"},
		{"127.0.0.1", "--concurrency", "1025", "--summary"},
		{"127.0.0.1", "--width", "16385", "--summary"},
		{"127.0.0.1", "--refresh", "1ms", "--summary"},
	}
	for _, args := range tests {
		cmd, _ := newRootCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "INPUT_OUT_OF_RANGE") {
			t.Errorf("Execute(%v) error = %v, want INPUT_OUT_OF_RANGE", args, err)
		}
	}
}

func TestOutputFlagsRegistration(t *testing.T) {
	cmd, flags := newRootCmd()

	expectedFlags := []string{"ascii", "json", "summary", "html", "iframe", "png", "pings", "list", "examples", "interface", "interfaces"}
	for _, flagName := range expectedFlags {
		f := cmd.Flags().Lookup(flagName)
		if f == nil {
			t.Errorf("Expected flag --%s to be registered", flagName)
		}
	}

	// Verify shorthand -i for interface and -I for interfaces
	if f := cmd.Flags().Lookup("interface"); f == nil || f.Shorthand != "i" {
		t.Errorf("Expected shorthand -i for interface")
	}
	if f := cmd.Flags().Lookup("interfaces"); f == nil || f.Shorthand != "I" {
		t.Errorf("Expected shorthand -I for interfaces")
	}

	// Verify alias --embed and --text exist
	if cmd.Flags().Lookup("embed") == nil {
		t.Errorf("Expected alias --embed to be registered")
	}
	if cmd.Flags().Lookup("text") == nil {
		t.Errorf("Expected alias --text to be registered")
	}

	// Verify shorthand -l exists for list
	listFlag := cmd.Flags().Lookup("list")
	if listFlag == nil || listFlag.Shorthand != "l" {
		t.Errorf("Expected shorthand -l to be registered for list")
	}

	// Verify shorthand -v is for verbose, NOT version
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil || verboseFlag.Shorthand != "v" {
		t.Errorf("Expected shorthand -v to be registered for verbose")
	}
	versionFlag := cmd.Flags().Lookup("version")
	if versionFlag == nil || versionFlag.Shorthand != "" {
		t.Errorf("Expected flag --version to have NO shorthand, got %q", versionFlag.Shorthand)
	}
	if cmd.Flags().Lookup("ver") == nil {
		t.Errorf("Expected flag --ver to be registered")
	}

	// Test removed flags: -f/--output-file, -t/--target, -o/--output, --redact, --yes, --no-input, --fast-threshold, --gateway
	if cmd.Flags().Lookup("output-file") != nil {
		t.Error("Expected flag -f/--output-file to be removed")
	}
	if cmd.Flags().Lookup("target") != nil {
		t.Error("Expected flag -t/--target to be removed")
	}
	if cmd.PersistentFlags().Lookup("output") != nil {
		t.Error("Expected flag -o/--output to be removed")
	}
	if cmd.PersistentFlags().Lookup("redact") != nil {
		t.Error("Expected flag --redact to be removed")
	}
	if cmd.PersistentFlags().Lookup("yes") != nil {
		t.Error("Expected flag --yes to be removed")
	}
	if cmd.PersistentFlags().Lookup("no-input") != nil {
		t.Error("Expected flag --no-input to be removed")
	}
	if cmd.Flags().Lookup("fast-threshold") != nil {
		t.Error("Expected flag --fast-threshold to be removed")
	}
	if cmd.Flags().Lookup("gateway") != nil {
		t.Error("Expected flag --gateway to be removed")
	}
	if cmd.PersistentFlags().Lookup("ps") != nil {
		t.Error("Expected flag --ps to be removed")
	}

	_ = flags
}

func TestPingsFlag(t *testing.T) {
	// Test default pings is 3
	cmdDef, flagsDef := newRootCmd()
	cmdDef.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--summary", "--quiet"}))
	if err := cmdDef.Execute(); err != nil {
		t.Fatalf("Default command failed: %v", err)
	}
	if flagsDef.pings != 3 {
		t.Errorf("Expected default flags.pings to be 3, got %d", flagsDef.pings)
	}

	cmd, flags := newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "-p", "2", "--summary", "--quiet"}))

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed with -p 2: %v", err)
	}
	if flags.pings != 2 {
		t.Errorf("Expected flags.pings to be 2, got %d", flags.pings)
	}

	// Test invalid pings <= 0
	cmd2, _ := newRootCmd()
	cmd2.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "-p", "0", "--summary"}))
	err = cmd2.Execute()
	if err == nil {
		t.Fatal("Expected error with -p 0, got nil")
	}
	if !strings.Contains(err.Error(), "INPUT_OUT_OF_RANGE") {
		t.Errorf("Expected INPUT_OUT_OF_RANGE error, got: %v", err)
	}
}

func TestNormalizer(t *testing.T) {
	// Space-separated filename should be merged into --html=file
	args := normalizeCLIArgs([]string{"pg", "127.0.0.1", "--html", "mygrid.html"})
	expected := []string{"pg", "127.0.0.1", "--html=mygrid.html"}
	if len(args) != len(expected) || args[2] != expected[2] {
		t.Errorf("normalizeCLIArgs failed, expected %v, got %v", expected, args)
	}

	// A filename before the target is ambiguous and must not be consumed.
	args = normalizeCLIArgs([]string{"pg", "--html", "mygrid.html", "127.0.0.1"})
	expected = []string{"pg", "--html", "mygrid.html", "127.0.0.1"}
	if len(args) != len(expected) || args[1] != expected[1] || args[2] != expected[2] || args[3] != expected[3] {
		t.Errorf("normalizeCLIArgs failed, expected %v, got %v", expected, args)
	}

	// Target after --html should NOT be merged
	args = normalizeCLIArgs([]string{"pg", "--html", "127.0.0.1"})
	expected = []string{"pg", "--html", "127.0.0.1"}
	if len(args) != len(expected) || args[1] != expected[1] || args[2] != expected[2] {
		t.Errorf("normalizeCLIArgs failed, expected %v, got %v", expected, args)
	}

	// CIDR target after --html should NOT be merged
	args = normalizeCLIArgs([]string{"pg", "--html", "192.168.1.0/24"})
	expected = []string{"pg", "--html", "192.168.1.0/24"}
	if len(args) != len(expected) || args[1] != expected[1] || args[2] != expected[2] {
		t.Errorf("normalizeCLIArgs failed, expected %v, got %v", expected, args)
	}

	// Hostnames and wildcard expressions after an output flag are targets, not files.
	for _, target := range []string{"printer.local", "192.168.1.*", "192.168.1.[1-30]"} {
		args = normalizeCLIArgs([]string{"pg", "--json", target})
		if len(args) != 3 || args[1] != "--json" || args[2] != target {
			t.Errorf("normalizeCLIArgs consumed target %q as a file: %v", target, args)
		}
	}

	// Space-separated filename for --list and -l
	args = normalizeCLIArgs([]string{"pg", "127.0.0.1", "--list", "hosts.txt"})
	expected = []string{"pg", "127.0.0.1", "--list=hosts.txt"}
	if len(args) != len(expected) || args[2] != expected[2] {
		t.Errorf("normalizeCLIArgs failed for --list, expected %v, got %v", expected, args)
	}

	args = normalizeCLIArgs([]string{"pg", "127.0.0.1", "-l", "hosts.txt"})
	expected = []string{"pg", "127.0.0.1", "-l=hosts.txt"}
	if len(args) != len(expected) || args[2] != expected[2] {
		t.Errorf("normalizeCLIArgs failed for -l, expected %v, got %v", expected, args)
	}

	// Non-output flag like -p should not be affected
	args = normalizeCLIArgs([]string{"pg", "127.0.0.1", "-p", "3"})
	expected = []string{"pg", "127.0.0.1", "-p", "3"}
	if len(args) != len(expected) || args[2] != expected[2] || args[3] != expected[3] {
		t.Errorf("normalizeCLIArgs failed, expected %v, got %v", expected, args)
	}

	// Test flexible prefix support: -, --, /
	switchTests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"pg", "/html", "out.html", "127.0.0.1"}, []string{"pg", "--html", "out.html", "127.0.0.1"}},
		{[]string{"pg", "-html", "out.html", "127.0.0.1"}, []string{"pg", "--html", "out.html", "127.0.0.1"}},
		{[]string{"pg", "/p", "3", "127.0.0.1"}, []string{"pg", "-p", "3", "127.0.0.1"}},
		{[]string{"pg", "--p", "3", "127.0.0.1"}, []string{"pg", "-p", "3", "127.0.0.1"}},
		{[]string{"pg", "/pings", "3", "127.0.0.1"}, []string{"pg", "--pings", "3", "127.0.0.1"}},
		{[]string{"pg", "-pings", "3", "127.0.0.1"}, []string{"pg", "--pings", "3", "127.0.0.1"}},
		{[]string{"pg", "/p:3", "127.0.0.1"}, []string{"pg", "-p=3", "127.0.0.1"}},
		{[]string{"pg", "/rows:5", "127.0.0.1"}, []string{"pg", "--rows=5", "127.0.0.1"}},
		{[]string{"pg", "/r", "5", "127.0.0.1"}, []string{"pg", "-r", "5", "127.0.0.1"}},
		{[]string{"pg", "--r", "5", "127.0.0.1"}, []string{"pg", "-r", "5", "127.0.0.1"}},
		{[]string{"pg", "/c", "10", "127.0.0.1"}, []string{"pg", "-c", "10", "127.0.0.1"}},
		{[]string{"pg", "/cols", "10", "127.0.0.1"}, []string{"pg", "--cols", "10", "127.0.0.1"}},
		{[]string{"pg", "/s", "moss", "127.0.0.1"}, []string{"pg", "-s", "moss", "127.0.0.1"}},
		{[]string{"pg", "--s", "moss", "127.0.0.1"}, []string{"pg", "-s", "moss", "127.0.0.1"}},
		{[]string{"pg", "/scheme:earth", "127.0.0.1"}, []string{"pg", "--scheme=earth", "127.0.0.1"}},
		{[]string{"pg", "/v", "127.0.0.1"}, []string{"pg", "-v", "127.0.0.1"}},
		{[]string{"pg", "--v", "127.0.0.1"}, []string{"pg", "-v", "127.0.0.1"}},
		{[]string{"pg", "/verbose", "127.0.0.1"}, []string{"pg", "-v", "127.0.0.1"}},
		{[]string{"pg", "/plain", "127.0.0.1"}, []string{"pg", "--plain", "127.0.0.1"}},
		{[]string{"pg", "-plain", "127.0.0.1"}, []string{"pg", "--plain", "127.0.0.1"}},
		{[]string{"pg", "/?"}, []string{"pg", "--help"}},
		{[]string{"pg", "-?"}, []string{"pg", "--help"}},
		{[]string{"pg", "/h"}, []string{"pg", "--help"}},
		{[]string{"pg", "-h"}, []string{"pg", "--help"}},
		{[]string{"pg", "/help"}, []string{"pg", "--help"}},
		{[]string{"pg", "/ver"}, []string{"pg", "--version"}},
		{[]string{"pg", "/version"}, []string{"pg", "--version"}},
		{[]string{"pg", "-i", "eth0"}, []string{"pg", "-i", "eth0"}},
		{[]string{"pg", "/i", "eth0"}, []string{"pg", "-i", "eth0"}},
		{[]string{"pg", "/i:eth0"}, []string{"pg", "-i=eth0"}},
		{[]string{"pg", "--interface", "eth0"}, []string{"pg", "--interface", "eth0"}},
		{[]string{"pg", "/interface", "eth0"}, []string{"pg", "--interface", "eth0"}},
		{[]string{"pg", "-I"}, []string{"pg", "-I"}},
		{[]string{"pg", "/I"}, []string{"pg", "-I"}},
		{[]string{"pg", "--interfaces"}, []string{"pg", "--interfaces"}},
		{[]string{"pg", "/interfaces"}, []string{"pg", "--interfaces"}},
		{[]string{"pg", "interfaces"}, []string{"pg", "--interfaces"}},
	}

	for _, tc := range switchTests {
		res := normalizeCLIArgs(tc.input)
		if len(res) != len(tc.expected) {
			t.Errorf("normalizeCLIArgs(%v) length = %d, want %d: %v", tc.input, len(res), len(tc.expected), res)
			continue
		}
		for idx := range res {
			if res[idx] != tc.expected[idx] {
				t.Errorf("normalizeCLIArgs(%v)[%d] = %q, want %q", tc.input, idx, res[idx], tc.expected[idx])
			}
		}
	}
}

func TestHelpOutputGrouping(t *testing.T) {
	cmd, _ := newRootCmd()
	help := buildHelpText(cmd)

	outIdx := strings.Index(help, "Output Options:")
	scanIdx := strings.Index(help, "Scan Options:")
	gridIdx := strings.Index(help, "Grid Layout Options:")
	colorIdx := strings.Index(help, "Color & Styling Options:")
	stdIdx := strings.Index(help, "Standard Options:")

	if outIdx == -1 || scanIdx == -1 || gridIdx == -1 || colorIdx == -1 || stdIdx == -1 {
		t.Fatalf("Help text missing required group headers:\n%s", help)
	}

	if outIdx >= scanIdx || scanIdx >= gridIdx || gridIdx >= colorIdx || colorIdx >= stdIdx {
		t.Errorf("Help groups not in expected order (Output first, Scan second, etc.): out=%d, scan=%d, grid=%d, color=%d, std=%d",
			outIdx, scanIdx, gridIdx, colorIdx, stdIdx)
	}

	// Verify --output and --target do not appear in help text
	if strings.Contains(help, "--output-file") || strings.Contains(help, "--target") {
		t.Errorf("Help text should not contain removed flags --output-file or --target")
	}
}

func TestOutputFlagsConflict(t *testing.T) {
	cmd, _ := newRootCmd()
	cmd.SetArgs([]string{"127.0.0.1", "--json", "--html"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when multiple conflicting output flags are specified, got nil")
	}
	if !strings.Contains(err.Error(), "USAGE_CONFLICTING_FLAGS") {
		t.Errorf("Expected USAGE_CONFLICTING_FLAGS error, got: %v", err)
	}
}

func TestOutputFlagsTripleConflict(t *testing.T) {
	cmd, _ := newRootCmd()
	cmd.SetArgs([]string{"127.0.0.1", "--ascii", "--json", "--summary"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when multiple conflicting output flags are specified, got nil")
	}
	if !strings.Contains(err.Error(), "USAGE_CONFLICTING_FLAGS") {
		t.Errorf("Expected USAGE_CONFLICTING_FLAGS error, got: %v", err)
	}
}

func TestGridCapacityExceeded(t *testing.T) {
	cmd, _ := newRootCmd()
	// 192.168.1.0/28 has 16 hosts; -r 2 -c 2 only has 4 slots
	cmd.SetArgs([]string{"192.168.1.0/28", "-r", "2", "-c", "2", "--summary"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when IP range exceeds grid capacity, got nil")
	}
	if !strings.Contains(err.Error(), "GRID_CAPACITY_EXCEEDED") {
		t.Errorf("Expected GRID_CAPACITY_EXCEEDED error, got: %v", err)
	}
}

func TestGridAutoLayoutSingleHost(t *testing.T) {
	cmd, flags := newRootCmd()
	cmd.SetArgs([]string{"127.0.0.1", "--summary", "--quiet"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if flags.rows != 1 || flags.cols != 1 {
		t.Errorf("Expected 1x1 grid for single host, got %dx%d", flags.rows, flags.cols)
	}
}

func TestGridAutoLayout50Hosts(t *testing.T) {
	cmd, flags := newRootCmd()
	cmd.SetArgs([]string{"127.0.0.1-50", "--summary", "--quiet"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if flags.rows != 5 || flags.cols != 10 {
		t.Errorf("Expected 5x10 grid for 50 hosts, got %dx%d", flags.rows, flags.cols)
	}
}

func TestOutputFilesAcceptFilename(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Test --html [file]
	htmlPath := filepath.Join(tempDir, "test.html")
	cmd, _ := newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--html", htmlPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--html file failed: %v", err)
	}
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Errorf("Expected HTML file %s to be created", htmlPath)
	}

	// 2. Test --json [file]
	jsonPath := filepath.Join(tempDir, "test.json")
	cmd, _ = newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--json", jsonPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--json file failed: %v", err)
	}
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("Expected JSON file %s to be created", jsonPath)
	}

	// 3. Test --summary [file]
	summaryPath := filepath.Join(tempDir, "test.txt")
	cmd, _ = newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--summary", summaryPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--summary file failed: %v", err)
	}
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Errorf("Expected summary file %s to be created", summaryPath)
	}

	// 4. Test --ascii [file]
	textPath := filepath.Join(tempDir, "grid.txt")
	cmd, _ = newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--ascii", textPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--ascii file failed: %v", err)
	}
	if _, err := os.Stat(textPath); os.IsNotExist(err) {
		t.Errorf("Expected ascii text file %s to be created", textPath)
	}

	// 5. Test --png [file]
	pngPath := filepath.Join(tempDir, "test.png")
	cmd, _ = newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--png", pngPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--png file failed: %v", err)
	}
	if _, err := os.Stat(pngPath); os.IsNotExist(err) {
		t.Errorf("Expected PNG file %s to be created", pngPath)
	}

	// 6. Test -l / --list [file]
	listPath := filepath.Join(tempDir, "test_list.txt")
	cmd, _ = newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--list", listPath, "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--list file failed: %v", err)
	}
	if _, err := os.Stat(listPath); os.IsNotExist(err) {
		t.Errorf("Expected list file %s to be created", listPath)
	}
	content, err := os.ReadFile(listPath)
	if err != nil {
		t.Fatalf("Failed to read list file: %v", err)
	}
	if !strings.Contains(string(content), "HOST") || !strings.Contains(string(content), "127.0.0.1") {
		t.Errorf("List file missing expected content:\n%s", string(content))
	}
}

func TestListOutputFormat(t *testing.T) {
	cmd, flags := newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "-l", "--quiet"}))

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed with -l: %v", err)
	}
	if flags.outputFormat != "list" {
		t.Errorf("Expected outputFormat to be 'list', got %q", flags.outputFormat)
	}
}

func TestExamplesFlag(t *testing.T) {
	cmd, _ := newRootCmd()
	cmd.SetArgs([]string{"--examples"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed with --examples: %v", err)
	}

	examples := buildExamplesText()
	requiredSnippets := []string{
		"Supported IP Target Range Formats:",
		"192.168.1.50",
		"192.168.1.0/24",
		"192.168.1.1-50",
		"192.168.1.100-192.168.1.200",
		"192.168.1.*",
		"192.168.1.[1-30]",
		"192.168.1.0/255.255.255.0",
		"192.168.1.1,192.168.1.254,10.0.0.1-10",
		"Output Formats:",
		"Scan Configuration & Tuning:",
		"Color Themes & Palette Customization:",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(examples, snippet) {
			t.Errorf("Examples text missing required snippet %q", snippet)
		}
	}
}

func TestVersionAndVerboseFlags(t *testing.T) {
	// 1. Test -v triggers Verbose mode, not version
	cmd, flags := newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "-v", "--summary", "--quiet"}))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed with -v: %v", err)
	}
	if !flags.Verbose {
		t.Errorf("Expected -v to set flags.Verbose to true")
	}
	if flags.showVersion {
		t.Errorf("Expected -v to NOT set flags.showVersion")
	}

	// 2. Test --ver triggers showVersion
	cmdVer, flagsVer := newRootCmd()
	cmdVer.SetArgs(normalizeCLIArgs([]string{"--ver"}))
	if err := cmdVer.Execute(); err != nil {
		t.Fatalf("Command failed with --ver: %v", err)
	}
	if !flagsVer.showVersion {
		t.Errorf("Expected --ver to set flags.showVersion to true")
	}

	// 3. Test --version triggers showVersion
	cmdVersion, flagsVersion := newRootCmd()
	cmdVersion.SetArgs(normalizeCLIArgs([]string{"--version"}))
	if err := cmdVersion.Execute(); err != nil {
		t.Fatalf("Command failed with --version: %v", err)
	}
	if !flagsVersion.showVersion {
		t.Errorf("Expected --version to set flags.showVersion to true")
	}
}

func TestShellCompletionSubcommands(t *testing.T) {
	cmd, _ := newRootCmd()
	completionCmd, _, err := cmd.Find([]string{"completion"})
	if err != nil || completionCmd == nil {
		t.Fatalf("Expected 'completion' command to exist")
	}

	shells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range shells {
		subCmd, _, err := cmd.Find([]string{"completion", shell})
		if err != nil || subCmd == nil || subCmd.Name() != shell {
			t.Errorf("Expected completion subcommand for shell %q", shell)
		}
	}
}

func TestARPCacheRemoved(t *testing.T) {
	cmd, _ := newRootCmd()
	for _, flagName := range []string{"arp-cache", "arp_cache", "use-arp-cache"} {
		if f := cmd.Flags().Lookup(flagName); f != nil {
			t.Errorf("Expected legacy --%s to be removed, but it was found", flagName)
		}
	}

	// Executing with removed legacy --arp-cache should fail with unknown flag error
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--arp-cache"}))
	if err := cmd.Execute(); err == nil {
		t.Errorf("Expected error when running with removed legacy --arp-cache flag, but got nil")
	}
}

func TestDefaultConcurrency(t *testing.T) {
	_, flags := newRootCmd()
	if flags.concurrency != 256 {
		t.Errorf("Expected default concurrency to be 256, got %d", flags.concurrency)
	}
}

func TestOptimizeOS(t *testing.T) {
	// 1. Flag existence
	cmd, _ := newRootCmd()
	if f := cmd.Flags().Lookup("optimize-os"); f == nil {
		t.Fatal("Expected --optimize-os flag to be registered on root command")
	}
	if f := cmd.Flags().Lookup("dry-run"); f == nil {
		t.Fatal("Expected --dry-run flag to be registered on root command")
	}

	// 2. Subcommand existence
	var subCmdFound bool
	for _, c := range cmd.Commands() {
		if c.Name() == "optimize-os" {
			subCmdFound = true
			break
		}
	}
	if !subCmdFound {
		t.Fatal("Expected optimize-os subcommand to be registered")
	}

	// 3. Dry-run execution via flag
	cmdFlag, _ := newRootCmd()
	cmdFlag.SetArgs(normalizeCLIArgs([]string{"--optimize-os", "--dry-run"}))
	if err := cmdFlag.Execute(); err != nil {
		t.Fatalf("Failed to execute --optimize-os --dry-run: %v", err)
	}

	// 4. Dry-run execution via subcommand
	cmdSub, _ := newRootCmd()
	cmdSub.SetArgs(normalizeCLIArgs([]string{"optimize-os", "--dry-run"}))
	if err := cmdSub.Execute(); err != nil {
		t.Fatalf("Failed to execute optimize-os --dry-run: %v", err)
	}

	// 5. Windows slash syntax /optimize-os /dry-run
	cmdSlash, _ := newRootCmd()
	cmdSlash.SetArgs(normalizeCLIArgs([]string{"/optimize-os", "/dry-run"}))
	if err := cmdSlash.Execute(); err != nil {
		t.Fatalf("Failed to execute /optimize-os /dry-run: %v", err)
	}
}

func TestInterfaceListingExecution(t *testing.T) {
	// Table mode
	cmd1, _ := newRootCmd()
	cmd1.SetArgs(normalizeCLIArgs([]string{"--interfaces"}))
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("Failed to execute --interfaces: %v", err)
	}

	// JSON mode
	cmd2, _ := newRootCmd()
	cmd2.SetArgs(normalizeCLIArgs([]string{"--interfaces", "--json"}))
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("Failed to execute --interfaces --json: %v", err)
	}

	// Shorthand -I
	cmd3, _ := newRootCmd()
	cmd3.SetArgs(normalizeCLIArgs([]string{"-I"}))
	if err := cmd3.Execute(); err != nil {
		t.Fatalf("Failed to execute -I: %v", err)
	}
}

func TestVerboseImportantHostsSweep(t *testing.T) {
	var buf bytes.Buffer
	logger := &log.Logger{
		Stdout: &buf,
		Stderr: &buf,
		Quiet:  false,
	}

	appFl := &appFlags{
		Verbose:     true,
		target:      "127.0.0.1",
		concurrency: 2,
		pings:       1,
		optASCII:    stdoutSentinel,
	}

	scanCfg := scanner.Config{
		LocalHostIP: net.ParseIP("127.0.0.1"),
		GatewayIP:   net.ParseIP("192.0.2.1"), // RFC 5737 TEST-NET-1 (unreachable/offline)
		Timeout:     50 * time.Millisecond,
		Pings:       1,
	}

	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.0.2.1")}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := executeSweepIteration(ctx, appFl, grid.GridConfig{}, ips, scanCfg, logger, 1, nil)
	if err != nil {
		t.Fatalf("executeSweepIteration failed: %v", err)
	}

	output := buf.String()

	// 1. Verify online important host (127.0.0.1 [Me]) has + prefix and role badge
	if !strings.Contains(output, "+ Host 127.0.0.1 [Me] responded") {
		t.Errorf("Expected '+ Host 127.0.0.1 [Me] responded' in verbose output, got:\n%s", output)
	}

	// 2. Verify offline important host (192.0.2.1 [Gateway]) has + prefix, role badge, and offline message
	if !strings.Contains(output, "+ Host 192.0.2.1 [Gateway] did not respond (offline)") {
		t.Errorf("Expected '+ Host 192.0.2.1 [Gateway] did not respond (offline)' in verbose output, got:\n%s", output)
	}
}

func TestHUDFlagsAndBanner(t *testing.T) {
	// 1. Test default hud is true
	cmdDef, flagsDef := newRootCmd()
	cmdDef.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--summary", "--quiet"}))
	if err := cmdDef.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !flagsDef.showHUD {
		t.Errorf("Expected flags.showHUD to default to true")
	}

	// 2. Test --no-hud sets showHUD to false
	cmdNoHUD, flagsNoHUD := newRootCmd()
	cmdNoHUD.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--no-hud", "--summary", "--quiet"}))
	if err := cmdNoHUD.Execute(); err != nil {
		t.Fatalf("Command failed with --no-hud: %v", err)
	}
	if flagsNoHUD.showHUD {
		t.Errorf("Expected --no-hud to set flags.showHUD to false")
	}

	// 3. Test HUD banner output in executeSweepIteration
	var buf bytes.Buffer
	logger := &log.Logger{Stdout: &buf, Stderr: &buf}
	appFl := &appFlags{
		showHUD:  true,
		optASCII: stdoutSentinel,
	}
	gridCfg := grid.GridConfig{
		InterfaceName: "Ethernet 2",
		LinkHealth: &scanner.LinkHealth{
			AdapterModel: "Test Adapter",
			LinkSpeedStr: "1 Gbps",
			MTU:          1500,
		},
	}
	scanCfg := scanner.Config{
		LocalHostIP: net.ParseIP("127.0.0.1"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := executeSweepIteration(ctx, appFl, gridCfg, []net.IP{net.ParseIP("127.0.0.1")}, scanCfg, logger, 1, nil)
	if err != nil {
		t.Fatalf("executeSweepIteration failed: %v", err)
	}
	// Note that fmt.Print writes to stdout; verify execution without error
}

func TestARPFlagsAndSweep(t *testing.T) {
	// 1. Test default --arp is true
	cmdDefault, flagsDefault := newRootCmd()
	cmdDefault.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--summary", "--quiet"}))
	if err := cmdDefault.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !flagsDefault.enableARP {
		t.Errorf("Expected enableARP to default to true")
	}

	// 2. Test --no-arp sets enableARP to false
	cmdNoARP, flagsNoARP := newRootCmd()
	cmdNoARP.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--no-arp", "--summary", "--quiet"}))
	if err := cmdNoARP.Execute(); err != nil {
		t.Fatalf("Command failed with --no-arp: %v", err)
	}
	if flagsNoARP.enableARP {
		t.Errorf("Expected --no-arp to set flags.enableARP to false")
	}
}

func TestAllInterfacesFlags(t *testing.T) {
	cmd, flags := newRootCmd()
	f := cmd.Flags().Lookup("all-interfaces")
	if f == nil {
		t.Fatal("Expected --all-interfaces to be registered")
	}
	if f.Shorthand != "A" {
		t.Errorf("Expected shorthand -A for --all-interfaces, got %q", f.Shorthand)
	}

	// Test flag parsing
	cmd2, flags2 := newRootCmd()
	cmd2.SetArgs(normalizeCLIArgs([]string{"-A", "--summary", "--quiet"}))
	_ = cmd2.ParseFlags(normalizeCLIArgs([]string{"-A"}))
	if !flags2.allInterfaces {
		// Verify normalization
		norm, _ := normalizeFlagToken("-A")
		if norm != "--all-interfaces" {
			t.Errorf("Expected -A to normalize to --all-interfaces, got %s", norm)
		}
	}

	// Test conflicting with -i
	cmdConflict, _ := newRootCmd()
	cmdConflict.SetArgs(normalizeCLIArgs([]string{"--all-interfaces", "-i", "eth0"}))
	err := cmdConflict.Execute()
	if err == nil {
		t.Error("Expected error when combining --all-interfaces with -i")
	} else if !strings.Contains(err.Error(), "USAGE_CONFLICTING_FLAGS") {
		t.Errorf("Expected USAGE_CONFLICTING_FLAGS, got %v", err)
	}

	// Test conflicting with positional target
	cmdConflictTarget, _ := newRootCmd()
	cmdConflictTarget.SetArgs(normalizeCLIArgs([]string{"--all-interfaces", "10.8.0.0/24"}))
	err = cmdConflictTarget.Execute()
	if err == nil {
		t.Error("Expected error when combining --all-interfaces with positional target")
	} else if !strings.Contains(err.Error(), "USAGE_CONFLICTING_FLAGS") {
		t.Errorf("Expected USAGE_CONFLICTING_FLAGS, got %v", err)
	}
	_ = flags
}

func TestAllInterfacesExecution(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "multi-test.json")

	cmd, _ := newRootCmd()
	cmd.SetArgs(normalizeCLIArgs([]string{"--all-interfaces", "--json=" + jsonPath, "--timeout=50ms", "-p=1", "--quiet"}))
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute --all-interfaces sweep: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON output file: %v", err)
	}

	if !strings.Contains(string(data), `"multi_interface": true`) {
		t.Errorf("Expected 'multi_interface: true' in JSON output, got:\n%s", string(data))
	}
}
