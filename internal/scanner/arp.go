package scanner

import (
	"bufio"
	"bytes"
	"net"
	"strings"
)

// ARPEntry represents an entry in the system ARP / neighbor table.
type ARPEntry struct {
	IP  net.IP
	MAC string
}

// ReadARPCache reads the system ARP / neighbor cache and returns a map of IP string -> MAC address string.
// It is implemented natively per platform without external runtime dependencies.
func ReadARPCache() (map[string]string, error) {
	return readPlatformARPCache()
}

func parseProcNetARP(data []byte) map[string]string {
	table := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Skip header line
	if scanner.Scan() {
		_ = scanner.Text()
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			ipStr := fields[0]
			flags := fields[2]
			macStr := fields[3]
			// Flags: 0x0 is incomplete, 0x2 is complete
			if flags != "0x0" && macStr != "00:00:00:00:00:00" {
				if ip := net.ParseIP(ipStr); ip != nil {
					table[ip.String()] = macStr
				}
			}
		}
	}
	return table
}

func parseArpOutput(output string) map[string]string {
	table := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		var ipStr, macStr string
		for _, f := range fields {
			cleaned := strings.Trim(f, "()")
			if net.ParseIP(cleaned) != nil && ipStr == "" {
				ipStr = cleaned
			} else if strings.Count(f, ":") == 5 || strings.Count(f, "-") == 5 {
				macStr = strings.ToLower(strings.ReplaceAll(f, "-", ":"))
			}
		}
		if ipStr != "" && macStr != "" && macStr != "(incomplete)" {
			table[ipStr] = macStr
		}
	}
	return table
}
