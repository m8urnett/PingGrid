package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputFlagsRegistration(t *testing.T) {
	cmd, flags := newRootCmd()

	expectedFlags := []string{"ascii", "json", "summary", "html", "iframe", "png", "pings", "list", "examples"}
	for _, flagName := range expectedFlags {
		f := cmd.Flags().Lookup(flagName)
		if f == nil {
			t.Errorf("Expected flag --%s to be registered", flagName)
		}
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

	// Space-separated filename before target
	args = normalizeCLIArgs([]string{"pg", "--html", "mygrid.html", "127.0.0.1"})
	expected = []string{"pg", "--html=mygrid.html", "127.0.0.1"}
	if len(args) != len(expected) || args[1] != expected[1] || args[2] != expected[2] {
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
		{[]string{"pg", "/html", "out.html", "127.0.0.1"}, []string{"pg", "--html=out.html", "127.0.0.1"}},
		{[]string{"pg", "-html", "out.html", "127.0.0.1"}, []string{"pg", "--html=out.html", "127.0.0.1"}},
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
	if flags.OutputFormat != "list" {
		t.Errorf("Expected OutputFormat to be 'list', got %q", flags.OutputFormat)
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
	for _, flagName := range []string{"arp-cache", "arp_cache", "arp", "use-arp-cache"} {
		if f := cmd.Flags().Lookup(flagName); f != nil {
			t.Errorf("Expected --%s to be removed, but it was found", flagName)
		}
	}

	// Executing with --arp-cache should fail with unknown flag error
	cmd.SetArgs(normalizeCLIArgs([]string{"127.0.0.1", "--arp-cache"}))
	if err := cmd.Execute(); err == nil {
		t.Errorf("Expected error when running with removed --arp-cache flag, but got nil")
	}
}


