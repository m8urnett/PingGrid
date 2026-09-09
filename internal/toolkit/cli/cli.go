package cli

import (
	"strings"

	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
)

// NormalizeFlag converts a flag to strict kebab-case as required by naming.md.
func NormalizeFlag(flag string) (string, error) {
	if flag == "" {
		return "", errors.New(errors.ExitInput, "USAGE_INVALID_NAME", "flag cannot be empty", flag, "", nil)
	}

	// Strip leading dashes
	clean := strings.TrimLeft(flag, "-/")

	// Check for camelCase anti-pattern
	if strings.ToLower(clean) != clean && !strings.Contains(clean, "-") {
		return "", errors.New(errors.ExitInput, "USAGE_INVALID_NAME", "camelCase is not allowed", flag, "Use kebab-case", nil)
	}

	// Normalize to kebab-case (basic implementation)
	return strings.ToLower(clean), nil
}

// IsHelpAlias checks if the given string is a known help alias.
func IsHelpAlias(arg string) bool {
	aliases := []string{"-h", "--help", "-?", "/?", "/h", "/help", "help"}
	for _, alias := range aliases {
		if arg == alias {
			return true
		}
	}
	return false
}

// IsVersionAlias checks the standard offline version triggers.
func IsVersionAlias(arg string) bool {
	aliases := []string{"--version", "/version", "version", "ver"}
	for _, alias := range aliases {
		if arg == alias {
			return true
		}
	}
	return false
}
