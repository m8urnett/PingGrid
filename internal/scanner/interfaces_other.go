//go:build !windows

package scanner

import (
	"bufio"
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func getPlatformDefaultGateway(dest net.IP) (net.IP, int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	command := "ip"
	args := []string{"-4", "route", "show", "default"}
	if runtime.GOOS == "darwin" {
		command = "route"
		args = []string{"-n", "get", "default"}
	}
	out, err := exec.CommandContext(ctx, command, args...).Output()
	if err != nil {
		return nil, 0
	}
	fields := strings.Fields(string(out))
	var gateway net.IP
	var ifaceName string
	for i, field := range fields {
		key := strings.TrimSuffix(strings.ToLower(field), ":")
		if (key == "via" || key == "gateway") && i+1 < len(fields) {
			gateway = net.ParseIP(strings.TrimSpace(fields[i+1]))
		}
		if (key == "dev" || key == "interface") && i+1 < len(fields) {
			ifaceName = strings.TrimSpace(fields[i+1])
		}
	}
	if gateway == nil {
		return nil, 0
	}
	if iface, lookupErr := net.InterfaceByName(ifaceName); lookupErr == nil {
		return gateway, iface.Index
	}
	_ = dest
	return gateway, 0
}

func getPlatformDNSServers() []net.IP {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()

	var dnsIPs []net.IP
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ip := net.ParseIP(parts[1]).To4(); ip != nil && !ip.IsUnspecified() {
					dnsIPs = append(dnsIPs, ip)
				}
			}
		}
	}
	return dnsIPs
}

func getPlatformDHCPInfo(ifIndex int) (net.IP, bool) {
	return nil, false
}

func getPlatformLinkHealth(ifIndex int, ifName string, mtu int, dhcpEnabled bool, dhcpServer net.IP) LinkHealth {
	lh := LinkHealth{
		MTU:         mtu,
		DHCPEnabled: dhcpEnabled,
	}
	if dhcpEnabled {
		if dhcpServer != nil {
			lh.DHCPStatus = "Active (" + dhcpServer.String() + ")"
		} else {
			lh.DHCPStatus = "Active"
		}
	} else {
		lh.DHCPStatus = "Unknown"
	}

	// Linux sysfs speed & duplex
	if data, err := os.ReadFile("/sys/class/net/" + ifName + "/speed"); err == nil {
		s := strings.TrimSpace(string(data))
		var mbps uint64
		for _, c := range s {
			if c >= '0' && c <= '9' {
				mbps = mbps*10 + uint64(c-'0')
			}
		}
		if mbps > 0 {
			lh.LinkSpeedBps = mbps * 1_000_000
			lh.LinkSpeedStr = FormatSpeed(lh.LinkSpeedBps)
		}
	}

	if data, err := os.ReadFile("/sys/class/net/" + ifName + "/duplex"); err == nil {
		d := strings.ToLower(strings.TrimSpace(string(data)))
		if d == "full" {
			lh.Duplex = "Full Duplex"
		} else if d == "half" {
			lh.Duplex = "Half Duplex"
		}
	}

	return lh
}
