package app

import (
	"net"

	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/errors"
	"github.com/spf13/cobra"
)

type resolvedTarget struct {
	ips         []net.IP
	gateway     net.IP
	localHost   net.IP
	dnsServers  []net.IP
	dhcpServer  net.IP
	activeIface *scanner.InterfaceInfo
}

func resolveScanTarget(flags *appFlags) (resolvedTarget, error) {
	var resolved resolvedTarget

	if flags.target == "" {
		if flags.ifaceTarget != "" {
			iface, err := scanner.FindInterface(flags.ifaceTarget)
			if err != nil {
				return resolved, errors.New(errors.ExitInput, "INTERFACE_NOT_FOUND", "Specified network interface not found", flags.ifaceTarget, "Run 'pg -I' or 'pg --interfaces' to see available interfaces", err)
			}
			if iface.IP == nil {
				return resolved, errors.New(errors.ExitInput, "INTERFACE_NO_IPV4", "Specified network interface has no IPv4 address", iface.Name, "Choose an active interface with an assigned IPv4 address", nil)
			}
			flags.target = iface.SweepCIDR()
			resolved.gateway = iface.Gateway
			resolved.localHost = iface.IP
			resolved.dnsServers = iface.DNSServers
			resolved.dhcpServer = iface.DHCPServer
			resolved.activeIface = iface
		} else {
			topology, err := scanner.DetectLocalTopology()
			if err != nil {
				return resolved, errors.New(errors.ExitProcessing, "SUBNET_DETECT_FAILED", "Failed to detect local network subnet", "", "Specify a target CIDR explicitly (e.g. pg 192.168.1.0/24) or run 'pg --help'", err)
			}
			flags.target = topology.SweepCIDR()
			resolved.gateway = topology.Gateway
			resolved.localHost = topology.IP
			resolved.dnsServers = topology.DNSServers
			resolved.dhcpServer = topology.DHCPServer
			resolved.activeIface = &topology
		}
	}

	ips, generatedGateway, err := scanner.GenerateIPs(flags.target, 0)
	if err != nil {
		return resolved, errors.New(errors.ExitInput, "INPUT_INVALID", "Invalid target address or CIDR subnet", flags.target, "Provide a valid IPv4 CIDR (e.g. 192.168.1.0/24) or base IP", err)
	}
	resolved.ips = ips
	if resolved.gateway == nil {
		resolved.gateway = generatedGateway
	}

	if resolved.activeIface == nil && len(ips) > 0 {
		ifaces, err := scanner.GetNetworkInterfaces()
		if err == nil {
			for i := range ifaces {
				if ifaces[i].IP == nil || ifaces[i].CIDR == "" {
					continue
				}
				_, network, parseErr := net.ParseCIDR(ifaces[i].CIDR)
				if parseErr != nil || !network.Contains(ips[0]) {
					continue
				}
				resolved.localHost = ifaces[i].IP
				if resolved.gateway == nil {
					resolved.gateway = ifaces[i].Gateway
				}
				resolved.dnsServers = ifaces[i].DNSServers
				resolved.dhcpServer = ifaces[i].DHCPServer
				resolved.activeIface = &ifaces[i]
				break
			}
		}
	}

	return resolved, nil
}

func buildScannerConfig(cmd *cobra.Command, flags *appFlags, target string, ips []net.IP, gateway, localHost net.IP, dnsServers []net.IP, dhcpServer net.IP) scanner.Config {
	pingTimeout := scanner.ResolveDefaultTimeout(ips)
	if cmd.Flags().Changed("timeout") {
		pingTimeout = flags.Timeout
	}
	return scanner.Config{
		Count:         len(ips),
		Concurrency:   flags.concurrency,
		Timeout:       pingTimeout,
		SlowThreshold: flags.slowThreshold,
		GatewayIP:     gateway,
		LocalHostIP:   localHost,
		DNSServers:    dnsServers,
		DHCPServer:    dhcpServer,
		Pings:         flags.pings,
		BroadcastIPs:  scanner.ExtractBroadcastIPs(target),
		DisableARP:    !flags.enableARP,
	}
}
