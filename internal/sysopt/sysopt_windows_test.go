//go:build windows

package sysopt

import (
	"strings"
	"testing"
)

func TestFormatNICPropertyCommandEscapesPowerShellLiterals(t *testing.T) {
	command := formatNICPropertyCommand("Adapter'; Write-Host pwned; '", "Prop'Name", "Enabled")
	if strings.Contains(command, "Adapter'; Write-Host") {
		t.Fatalf("adapter name was inserted as executable PowerShell syntax: %s", command)
	}
	if !strings.Contains(command, "Adapter''; Write-Host pwned; ''") {
		t.Fatalf("adapter name was not escaped as a PowerShell literal: %s", command)
	}
}
