//go:build !windows

package scanner

import (
	"bytes"
	"os"
	"os/exec"
)

func readPlatformARPCache() (map[string]string, error) {
	// Try /proc/net/arp on Linux first
	if data, err := os.ReadFile("/proc/net/arp"); err == nil {
		return parseProcNetARP(data), nil
	}

	// Fallback to `arp -an` or `arp -a` on macOS/POSIX
	cmd := exec.Command("arp", "-an")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return parseArpOutput(out.String()), nil
	}

	cmdFallback := exec.Command("arp", "-a")
	out.Reset()
	cmdFallback.Stdout = &out
	if err := cmdFallback.Run(); err == nil {
		return parseArpOutput(out.String()), nil
	}

	return make(map[string]string), nil
}
