package log

import (
	"fmt"
	"io"
	"os"
)

// Logger abstracts the routing of diagnostic info vs data payloads.
type Logger struct {
	Stdout io.Writer
	Stderr io.Writer
	Quiet  bool
}

// New creates a new standard logger routing to os.Stdout and os.Stderr.
func New(quiet bool) *Logger {
	return &Logger{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Quiet:  quiet,
	}
}

// Data outputs raw data payload strictly to stdout.
func (l *Logger) Data(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(l.Stdout, format+"\n", args...)
	return err
}

// Diagnostic outputs diagnostic info to stderr. Honors the Quiet flag.
func (l *Logger) Diagnostic(format string, args ...interface{}) error {
	if !l.Quiet {
		_, err := fmt.Fprintf(l.Stderr, "[INFO] "+format+"\n", args...)
		return err
	}
	return nil
}

// Warn always outputs to stderr regardless of Quiet flag.
func (l *Logger) Warn(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(l.Stderr, "[WARN] "+format+"\n", args...)
	return err
}
