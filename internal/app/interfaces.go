package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m8urnett/PingGrid/internal/scanner"
	"github.com/m8urnett/toolkit/errors"
	"github.com/m8urnett/toolkit/log"
	"github.com/spf13/cobra"
)

func runListInterfaces(cmd *cobra.Command, flags *appFlags) error {
	logger := log.New(flags.Quiet)
	ifaces, err := scanner.GetNetworkInterfaces()
	if err != nil {
		return errors.New(errors.ExitProcessing, "INTERFACES_FAILED", "Failed to enumerate network interfaces", "", "Ensure system network stack is functioning", err)
	}

	if flags.optJSON != "" || flags.outputFormat == "json" {
		type jsonIface struct {
			Index      int                 `json:"index"`
			Name       string              `json:"name"`
			Model      string              `json:"model,omitempty"`
			LinkSpeed  string              `json:"link_speed,omitempty"`
			MAC        string              `json:"mac,omitempty"`
			IP         string              `json:"ip,omitempty"`
			CIDR       string              `json:"cidr,omitempty"`
			Gateway    string              `json:"gateway,omitempty"`
			MTU        int                 `json:"mtu"`
			Status     string              `json:"status"`
			IsPrimary  bool                `json:"primary"`
			LinkHealth *scanner.LinkHealth `json:"link_health,omitempty"`
		}
		var out []jsonIface
		for _, iface := range ifaces {
			status := "Down"
			if iface.IsUp {
				status = "Up"
			}
			ipStr := ""
			if iface.IP != nil {
				ipStr = iface.IP.String()
			}
			gwStr := ""
			if iface.Gateway != nil {
				gwStr = iface.Gateway.String()
			}
			out = append(out, jsonIface{
				Index:      iface.Index,
				Name:       iface.Name,
				Model:      iface.Health.AdapterModel,
				LinkSpeed:  iface.Health.LinkSpeedStr,
				MAC:        iface.HardwareAddr,
				IP:         ipStr,
				CIDR:       iface.CIDR,
				Gateway:    gwStr,
				MTU:        iface.MTU,
				Status:     status,
				IsPrimary:  iface.IsPrimary,
				LinkHealth: &iface.Health,
			})
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if flags.optJSON != stdoutSentinel && flags.optJSON != "" {
			return writeOutputFile(flags.optJSON, data)
		}
		_ = logger.Data("%s", string(data))
		return nil
	}

	var sb strings.Builder
	sb.WriteString("\nNetwork Interfaces:\n")
	fmt.Fprintf(&sb, "  %-5s %-25s %-6s %-18s %-10s %-17s %-15s %s\n",
		"INDEX", "NAME", "STATUS", "IP/CIDR", "SPEED", "MAC ADDRESS", "GATEWAY", "PRIMARY")
	fmt.Fprintf(&sb, "  %-5s %-25s %-6s %-18s %-10s %-17s %-15s %s\n",
		"-----", "-------------------------", "------", "------------------", "----------", "-----------------", "---------------", "-------")

	for _, iface := range ifaces {
		status := "Down"
		if iface.IsUp {
			status = "Up"
		}
		cidr := iface.CIDR
		if cidr == "" {
			cidr = "-"
		}
		mac := iface.HardwareAddr
		if mac == "" {
			mac = "-"
		}
		gw := "-"
		if iface.Gateway != nil {
			gw = iface.Gateway.String()
		}
		primary := ""
		if iface.IsPrimary {
			primary = "*"
		}
		speed := iface.Health.LinkSpeedStr
		if speed == "" {
			speed = "-"
		}

		name := iface.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		fmt.Fprintf(&sb, "  %-5d %-25s %-6s %-18s %-10s %-17s %-15s %s\n",
			iface.Index, name, status, cidr, speed, mac, gw, primary)
	}
	sb.WriteString("\n  * Primary interface used for default internet routing and auto-sweep\n\n")

	_ = logger.Data("%s", sb.String())
	return nil
}
