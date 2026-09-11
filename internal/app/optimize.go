package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/m8urnett/PingGrid/internal/sysopt"
	"github.com/m8urnett/toolkit/errors"
	"github.com/spf13/cobra"
)

func runOptimizeOS(cmd *cobra.Command, flags *appFlags) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), maxOperationTime)
	defer cancel()
	isPlain := plainConsoleOutput(flags)
	elevated := sysopt.IsElevated()

	fmt.Printf("PingGrid OS Network Optimizer (%s)\n", sysopt.OSName())
	fmt.Println(strings.Repeat("=", 65))

	if flags.dryRun {
		fmt.Println("Mode: DRY RUN (previewing proposed changes, none will be applied)")
	} else if elevated {
		fmt.Println("Privileges: Elevated (Administrator / Root)")
	} else {
		fmt.Println("Privileges: Standard User (Non-Elevated)")
	}
	fmt.Println()

	if !elevated && !flags.dryRun {
		opts, err := sysopt.Inspect(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Proposed System Optimizations:")
		for i, o := range opts {
			fmt.Printf("\n%d. %s\n", i+1, o.Name)
			fmt.Printf("   Description:   %s\n", o.Description)
			fmt.Printf("   Current Value: %s\n", o.CurrentValue)
			fmt.Printf("   Target Value:  %s\n", o.TargetValue)
			fmt.Printf("   Command:       %s\n", o.Command)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("-", 65))
		if isPlain {
			fmt.Println("[!] ATTENTION: Root/Administrator privileges are required to apply these changes.")
		} else {
			fmt.Println("\033[33m[!] ATTENTION: Root/Administrator privileges are required to apply these changes.\033[0m")
		}
		fmt.Printf("    %s\n", sysopt.ElevationInstructions())
		fmt.Println("    To preview without elevation, run with: --dry-run")
		return errors.New(errors.ExitConfiguration, "PRIVILEGE_REQUIRED", "OS optimizations were inspected but not applied", sysopt.OSName(), sysopt.ElevationInstructions(), nil)
	}

	results, err := sysopt.Apply(ctx, flags.dryRun)
	if err != nil {
		return err
	}

	var appliedCount, skippedCount, failedCount int
	for _, res := range results {
		badge := res.FormatBadge(isPlain)
		fmt.Printf("%s %s\n", badge, res.SummaryString())
		if res.Applied {
			appliedCount++
		} else if res.Skipped {
			skippedCount++
		} else if res.Err != nil {
			failedCount++
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 65))
	if flags.dryRun {
		fmt.Printf("Inspection complete: %d optimizations reviewed.\n", len(results))
		if !elevated {
			fmt.Printf("\nTo apply these optimizations: %s\n", sysopt.ElevationInstructions())
		}
	} else {
		fmt.Printf("Optimization complete: %d applied, %d already optimal, %d failed.\n",
			appliedCount, skippedCount, failedCount)
	}
	if failedCount > 0 {
		return errors.New(errors.ExitProcessing, "PARTIAL_MODIFICATION", "One or more OS optimizations failed", fmt.Sprintf("%d of %d steps failed", failedCount, len(results)), "Review the failed steps, restore affected settings if needed, and retry only after correcting the reported cause", nil)
	}

	return nil
}
