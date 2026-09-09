package exec

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
)

// Result describes a completed child process without merging its output streams.
type Result struct {
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Duration  time.Duration
	TimedOut  bool
	Cancelled bool
}

// Run executes a process directly and captures stdout and stderr independently.
func Run(ctx context.Context, name string, args ...string) (Result, error) {
	started := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Duration: time.Since(started)}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if ctx.Err() != nil {
		result.Cancelled = true
		result.TimedOut = ctx.Err() == context.DeadlineExceeded
	}
	if err != nil {
		return result, errors.New(errors.ExitProcessing, "EXEC_FAILED", "Failed to execute secure process", name, "Ensure the command is available and inspect stderr", err)
	}
	return result, nil
}

// RunSafe executes a command securely without invoking a shell interpreter.
func RunSafe(ctx context.Context, name string, args ...string) ([]byte, error) {
	result, err := Run(ctx, name, args...)
	if err != nil {
		return append(result.Stdout, result.Stderr...), err
	}
	return result.Stdout, nil
}
