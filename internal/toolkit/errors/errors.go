package errors

import (
	"fmt"
	"os"
)

// Standardized POSIX/Windows Exit Codes
const (
	ExitSuccess         = 0
	ExitFindings        = 1
	ExitUsage           = 2
	ExitInput           = 3
	ExitProcessing      = 4
	ExitOutput          = 5
	ExitConfiguration   = 6
	ExitDependency      = 7
	ExitInternal        = 8
	ExitParseData       = 9
	ExitPolicyViolation = 10
	ExitInterrupted     = 130
)

// Standardized Topic Codes
const (
	CodeInputNotFound            = "INPUT_NOT_FOUND"
	CodeParseFailed              = "PARSE_FAILED"
	CodeOutputWriteFailed        = "OUTPUT_WRITE_FAILED"
	CodeInternalError            = "INTERNAL_ERROR"
	CodeInterrupted              = "INTERRUPTED"
	CodePolicyViolation          = "POLICY_VIOLATION"
	CodeExitCodeResolutionFailed = "EXIT_CODE_RESOLUTION_FAILED"
)

// Error represents a standardized error adhering to findings-and-errors.md
type Error struct {
	ExitCode   int
	TopicCode  string
	Message    string
	Input      string
	Recovery   string
	Underlying error
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.TopicCode, e.Message)
	if e.Input != "" {
		msg += fmt.Sprintf(" (input: %s)", e.Input)
	}
	if e.Underlying != nil {
		msg += fmt.Sprintf(" - cause: %v", e.Underlying)
	}
	return msg
}

// Fatal prints the diagnostic error safely to stderr and exits with the stable code.
func Fatal(err error) {
	if e, ok := err.(*Error); ok {
		fmt.Fprintf(os.Stderr, "Error: %s\n", e.Error())
		if e.Recovery != "" {
			fmt.Fprintf(os.Stderr, "Recovery: %s\n", e.Recovery)
		}
		os.Exit(e.ExitCode)
	}

	// Fallback for non-standard errors
	fmt.Fprintf(os.Stderr, "Error: [%s] %v\n", CodeInternalError, err)
	os.Exit(ExitInternal)
}

// New creates a new standard error.
func New(exitCode int, topicCode, message, input, recovery string, underlying error) *Error {
	return &Error{
		ExitCode:   exitCode,
		TopicCode:  topicCode,
		Message:    message,
		Input:      input,
		Recovery:   recovery,
		Underlying: underlying,
	}
}
