package sysopt

import (
	"fmt"
	"runtime"
)

// Optimization describes a single host operating system network stack tuning.
type Optimization struct {
	ID           string
	Name         string
	Description  string
	CurrentValue string
	TargetValue  string
	Command      string
}

// StepResult represents the outcome of attempting to apply a single optimization.
type StepResult struct {
	Opt     Optimization
	Applied bool
	Skipped bool
	Message string
	Err     error
}

// OSName returns a user-friendly label for the current platform.
func OSName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	default:
		return runtime.GOOS
	}
}

// ElevationInstructions returns clear platform-specific guidance for obtaining root/admin rights.
func ElevationInstructions() string {
	switch runtime.GOOS {
	case "windows":
		return "Administrator privileges required. Open PowerShell or Command Prompt with 'Run as Administrator', then re-run this command."
	case "linux", "darwin":
		return "Root privileges required. Re-run this command using sudo (e.g., sudo pg --optimize-os)."
	default:
		return "Root or Administrator privileges required to modify operating system network parameters."
	}
}

// FormatBadge returns a color-safe or plain-text badge for the result.
func (sr StepResult) FormatBadge(isPlain bool) string {
	if sr.Skipped {
		if isPlain {
			return "[SKIP]"
		}
		return "\033[33m[SKIP]\033[0m"
	}
	if sr.Applied {
		if isPlain {
			return "[OK]  "
		}
		return "\033[32m[OK]  \033[0m"
	}
	if sr.Err != nil {
		if isPlain {
			return "[FAIL]"
		}
		return "\033[31m[FAIL]\033[0m"
	}
	return "[INFO]"
}

// SummaryString returns a human-readable description of the step outcome.
func (sr StepResult) SummaryString() string {
	if sr.Skipped {
		if sr.Message != "" {
			return fmt.Sprintf("%s: %s (Already optimal)", sr.Opt.Name, sr.Message)
		}
		return fmt.Sprintf("%s: Already optimal", sr.Opt.Name)
	}
	if sr.Applied {
		if sr.Message != "" {
			return fmt.Sprintf("%s: %s", sr.Opt.Name, sr.Message)
		}
		return fmt.Sprintf("%s: Successfully applied (%s)", sr.Opt.Name, sr.Opt.TargetValue)
	}
	if sr.Err != nil {
		return fmt.Sprintf("%s: Failed - %v", sr.Opt.Name, sr.Err)
	}
	return sr.Opt.Name
}
