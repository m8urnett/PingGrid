//go:build !windows

package scanner

import (
	"bufio"
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

func getPlatformARPTable() (map[string]ARPEntry, error) {
	result := make(map[string]ARPEntry)

	// 1. Linux: /proc/net/arp
	if f, err := os.Open("/proc/net/arp"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		isFirst := true
		for scanner.Scan() {
			if isFirst {
				isFirst = false // Skip header
				continue
			}
			fields := strings.Fields(scanner.Text())
			// IP address HW type Flags HW address Mask Device
			if len(fields) >= 4 {
				ip := net.ParseIP(fields[0])
				if ip == nil || ip.To4() == nil {
					continue
				}
				flags := fields[2]
				if flags == "0x0" { // Incomplete
					continue
				}
				mac := NormalizeMAC(fields[3])
				if !IsUnicastMAC(mac) {
					continue
				}
				ipStr := ip.String()
				result[ipStr] = ARPEntry{
					IP:       ip,
					MAC:      mac,
					Vendor:   LookupVendor(mac),
					IsStatic: flags == "0x6", // ATF_PERM | ATF_COM
				}
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// 2. macOS / BSD / Linux fallback: arp -an
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "arp", "-an")
	if out, err := cmd.Output(); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Format: ? (10.8.0.1) at 50:9a:4c:75:3e:2f on en0 ifscope [ethernet]
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				ipStr := strings.Trim(parts[1], "()")
				ip := net.ParseIP(ipStr)
				if ip == nil || ip.To4() == nil {
					continue
				}
				mac := NormalizeMAC(parts[3])
				if !IsUnicastMAC(mac) {
					continue
				}
				isStatic := strings.Contains(line, "permanent") || strings.Contains(line, "static")
				result[ipStr] = ARPEntry{
					IP:       ip,
					MAC:      mac,
					Vendor:   LookupVendor(mac),
					IsStatic: isStatic,
				}
			}
		}
	}

	return result, nil
}
