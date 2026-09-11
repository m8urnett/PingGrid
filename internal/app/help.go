package app

import (
	"github.com/spf13/cobra"
)

func buildHelpText(_ *cobra.Command) string {
	return `pg (PingGrid) performs a concurrent ping sweep across a target subnet or IP range
and renders an activity grid (terminal ASCII display, interactive HTML, or PNG image).

Usage:
  pg [target] [flags]

Output Options:
  -l, --list [file]              Output sorted list of host names and ping times (optionally write to file)
      --html [file]              Generate standalone interactive HTML dashboard (default: grid.html)
      --iframe [file]            Generate embeddable minimal HTML for iframes (default: grid-embed.html)
      --png [file]               Generate PNG activity grid image (default: grid.png)
      --json [file]              Output versioned machine-readable JSON (optionally write to file)
      --summary [file]           Output single-line text summary (optionally write to file)
      --ascii [file]             Output console ASCII text grid (optionally write to file)

Scan Options:
  -i, --interface <name|index>   Target network interface by name or index (e.g. 'Wi-Fi', 'eth0', 1)
  -I, --interfaces               List detected local network interfaces and exit
  -A, --all-interfaces           Sweep all active local network interfaces simultaneously
  -p, --pings <count>            Ping attempts per host, 1-10 (default 3)
  -R, --refresh <interval>       Refresh interval, 100ms-24h (0 runs once)
      --concurrency <workers>    Concurrent worker budget, 1-1024 (default 256)
      --timeout <duration>       Ping timeout, 1ms-1m (default 150ms LAN, 400ms WAN)
      --slow-threshold <duration> Latency threshold for slow/degraded color (default 100ms)
      --hud                      Display Interface & Link Health HUD banner (default true)
      --no-hud                   Disable Interface & Link Health HUD banner
      --arp                      Pre-warm discovery cache from OS neighbor table and detect silent hosts (default true)
      --no-arp                   Disable OS ARP/neighbor cache integration

Grid Layout Options:
  -r, --rows <count>             Number of grid rows (auto-sized to fit IP range if omitted)
  -c, --cols <count>             Number of grid columns (auto-sized to fit IP range if omitted)
  -W, --width <pixels>           Output image width in pixels (auto-sized if omitted)
  -H, --height <pixels>          Output image height in pixels (auto-sized if omitted)
      --border-width <pixels>    Grid divider border width in pixels (default 1)

Color & Styling Options:
  -s, --scheme <name>            Built-in color scheme (dark, light, earth, moss, linen) (default "dark")
      --color-offline <color>    Hex color for offline/unresponsive hosts (with or without #)
      --color-online <color>     Hex color for active/online hosts (with or without #)
      --color-highlight <color>  Hex color for fast/highlight hosts (with or without #)
      --color-slow <color>       Hex color for slow-responding hosts (with or without #)
      --color-border <color>     Hex color for cell divider lines (with or without #)
      --color-frame <color>      Hex color for outer canvas frame (with or without #)

Standard Options:
      --version, --ver           Display version information and exit
  -v, --verbose                  Enable detailed diagnostic output
  -q, --quiet                    Suppress non-essential console output
  -h, --help                     Display help and exit
      --examples                 Display detailed usage examples and target formats
      --plain                    Plain monochrome ASCII mode without ANSI colors
      --no-color                 Disable ANSI colors (also honored through NO_COLOR)

System Optimization:
      --optimize-os              Tune host OS network parameters for fast sweeping (requires admin/root)
      --dry-run                  Inspect proposed OS optimizations without applying changes

Examples:
  Run 'pg --examples' to view detailed usage examples and all supported IP range formats.

Shell Completion:
  Generate tab autocompletion script for commands, flags, and color schemes.

  Bash (load in current session):
    source <(pg completion bash)

  Zsh (load in current session):
    source <(pg completion zsh)

  PowerShell (load in current session):
    pg completion powershell | Out-String | Invoke-Expression

  PowerShell (persist across all sessions in $PROFILE):
    Add-Content $PROFILE "` + "`" + `\npg completion powershell | Out-String | Invoke-Expression` + "`" + `"

  Completes all flags (-s/--scheme, -p/--pings, --html, --png, --json, -R/--refresh, etc.), built-in
  color schemes (dark, light, earth, moss, linen), and options.
`
}

func buildExamplesText() string {
	return `PingGrid (pg) - Usage Examples & Target Range Formats

1. Supported IP Target Range Formats:
  # Single IPv4 address:
  pg 192.168.1.50

  # Standard CIDR subnet prefix:
  pg 192.168.1.0/24
  pg 10.0.0.0/28

  # Shorthand octet range:
  pg 192.168.1.1-50

  # Full IP start-end range:
  pg 192.168.1.100-192.168.1.200

  # Wildcard pattern (expands full octet 0-255):
  pg 192.168.1.*

  # Bracket range pattern across octets:
  pg 192.168.1.[1-30]
  pg 10.0.[1-2].[1-10]

  # Subnet with explicit netmask:
  pg 192.168.1.0/255.255.255.0

  # Comma-separated combination of addresses & ranges:
  pg 192.168.1.1,192.168.1.254,10.0.0.1-10

  # Auto-detected local subnet (omitting target scans active local subnet):
  pg

  # Target specific network interface by name or index:
  pg -i "Wi-Fi"
  pg -i 1

  # List all local network interfaces:
  pg -I
  pg --interfaces

2. Output Formats:
  # Sorted list of responding hosts (hostnames & ping times):
  pg 192.168.1.0/24 -l

  # Save sorted host list to file:
  pg 192.168.1.0/24 --list hosts.txt

  # Standalone interactive HTML dashboard (saves to grid.html):
  pg 192.168.1.0/24 --html

  # Standalone HTML dashboard with custom filename:
  pg 192.168.1.0/24 --html dashboard.html

  # Minimal embeddable HTML for iframes:
  pg 192.168.1.0/24 --iframe embed.html

  # PNG activity grid image (default: grid.png):
  pg 192.168.1.0/24 --png

  # PNG activity grid image with custom filename:
  pg 192.168.1.0/24 --png report.png

  # Machine-readable JSON summary (to stdout or file):
  pg 192.168.1.0/24 --json
  pg 192.168.1.0/24 --json scan.json

  # Single-line text summary:
  pg 192.168.1.0/24 --summary

  # Plain monochrome ASCII mode (no ANSI color escapes):
  pg 192.168.1.0/24 --plain

3. Scan Configuration & Tuning:
  # Multi-ping sweep (3 attempts per host, recording lowest RTT):
  pg 192.168.1.0/24 -p 3

  # Continuous live monitoring (refresh sweep every 5 seconds):
  pg 192.168.1.0/24 -R 5s

  # Multi-adapter simultaneous sweep across all active interfaces (LAN, Wi-Fi, Docker, WSL, VPN):
  pg -A
  pg --all-interfaces -l
  pg --all-interfaces --html=multi-dashboard.html

  # Adjust ping timeout (defaults to 150ms LAN, 400ms WAN):
  pg 10.0.0.0/24 --timeout 250ms

  # Set custom slow-response latency threshold:
  pg 192.168.1.0/24 --slow-threshold 50ms

  # Tune concurrency (worker pool size, default: 256):
  pg 10.0.0.0/16 --concurrency 512

4. Color Themes & Palette Customization:
  # Switch built-in scheme (dark, light, earth, moss, linen):
  pg 192.168.1.0/24 -s light
  pg 192.168.1.0/24 -s moss

  # Custom hex colors (with or without leading #):
  pg 192.168.1.0/24 --color-online 4d86a2 --color-offline 404e41
  pg 192.168.1.0/24 --color-online "#4d86a2" --color-slow "#ab7550"

5. Host Operating System Tuning:
  # Inspect proposed OS network optimizations (dry-run preview):
  pg optimize-os --dry-run

  # Apply OS network optimizations (run as Administrator / sudo):
  pg optimize-os
  pg --optimize-os
`
}
