package sysopt

import (
	"context"
	"strings"
	"testing"
)

func TestSysoptBasics(t *testing.T) {
	osName := OSName()
	if osName == "" {
		t.Error("Expected non-empty OS name")
	}

	instructions := ElevationInstructions()
	if instructions == "" {
		t.Error("Expected non-empty elevation instructions")
	}

	// Test badge formatting
	srOk := StepResult{Applied: true}
	if !strings.Contains(srOk.FormatBadge(true), "[OK]") {
		t.Errorf("Expected [OK] in plain badge, got %s", srOk.FormatBadge(true))
	}

	srSkip := StepResult{Skipped: true}
	if !strings.Contains(srSkip.FormatBadge(true), "[SKIP]") {
		t.Errorf("Expected [SKIP] in plain badge, got %s", srSkip.FormatBadge(true))
	}

	srFail := StepResult{Err: context.Canceled}
	if !strings.Contains(srFail.FormatBadge(true), "[FAIL]") {
		t.Errorf("Expected [FAIL] in plain badge, got %s", srFail.FormatBadge(true))
	}
}

func TestInspectAndDryRun(t *testing.T) {
	ctx := context.Background()

	opts, err := Inspect(ctx)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if len(opts) == 0 {
		t.Error("Expected at least one optimization defined for the platform")
	}

	// Dry run Apply should succeed without altering settings
	results, err := Apply(ctx, true)
	if err != nil {
		t.Fatalf("DryRun Apply failed: %v", err)
	}

	if len(results) != len(opts) {
		t.Errorf("Expected %d results from DryRun, got %d", len(opts), len(results))
	}

	for _, res := range results {
		if !res.Skipped {
			t.Errorf("Expected dry run result %q to be marked skipped, got applied=%v", res.Opt.Name, res.Applied)
		}
	}
}
