//go:build !windows && !linux && !darwin

package sysopt

import (
	"context"
	"fmt"
)

// IsElevated returns false on unsupported operating systems.
func IsElevated() bool {
	return false
}

// Inspect returns an empty list on unsupported platforms.
func Inspect(ctx context.Context) ([]Optimization, error) {
	return nil, fmt.Errorf("automated OS network optimization is not supported on this platform")
}

// Apply returns an unsupported error on other platforms.
func Apply(ctx context.Context, dryRun bool) ([]StepResult, error) {
	return nil, fmt.Errorf("automated OS network optimization is not supported on this platform")
}
