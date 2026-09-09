package main

import (
	"strings"
	"testing"
)

func TestOutputFlagsRegistration(t *testing.T) {
	cmd, flags := newRootCmd()

	expectedFlags := []string{"text", "json", "summary", "html", "iframe"}
	for _, flagName := range expectedFlags {
		f := cmd.Flags().Lookup(flagName)
		if f == nil {
			t.Errorf("Expected flag --%s to be registered", flagName)
		}
	}

	// Verify alias --embed exists
	if cmd.Flags().Lookup("embed") == nil {
		t.Errorf("Expected alias --embed to be registered")
	}

	// Test default output-file is empty (does not write grid files by default)
	outPathFlag := cmd.Flags().Lookup("output-file")
	if outPathFlag == nil {
		t.Fatal("Expected flag -f/--output-file to be registered")
	}
	if outPathFlag.DefValue != "" {
		t.Errorf("Expected default output-file to be empty, got: %q", outPathFlag.DefValue)
	}

	_ = flags
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
	cmd.SetArgs([]string{"127.0.0.1", "--text", "--json", "--summary"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when multiple conflicting output flags are specified, got nil")
	}
	if !strings.Contains(err.Error(), "USAGE_CONFLICTING_FLAGS") {
		t.Errorf("Expected USAGE_CONFLICTING_FLAGS error, got: %v", err)
	}
}
