# Changelog

All notable changes to PingGrid will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
PingGrid uses the project-specific padded `1.xx.NNN` application-version contract.

## [1.04.000] - 2026-09-11

### Changed
- Reduced the executable entry point to version declaration, application invocation, and fatal-error handling.
- Moved CLI construction, argument normalization, validation, sweep orchestration, reporting, interface inventory, and optimizer coordination into focused internal application components.
- Propagated the caller context through Cobra into single-interface, multi-interface, and optimizer operations.

## [1.03.000] - 2026-09-11

### Fixed
- Corrected the Windows `IP_ADAPTER_INFO` ABI layout, eliminating a race/checkptr crash while reading DHCP metadata.
- Replaced invalid POSIX UDP handling with `golang.org/x/net/icmp` datagram/raw ICMP sockets.
- Corrected JSON millisecond fields and added stable lower-case host telemetry.
- Stopped treating every `.255` address as a broadcast and stopped stale pre-scan ARP entries from promoting offline hosts.
- Corrected multi-interface ASCII file output, iframe rendering, fast filtering, grid validation, worker budgeting, and per-interface durations.
- Removed fabricated `.1` gateways, rejected ambiguous interface-name matches, and correlated Windows Wi-Fi details with the requested adapter.

### Changed
- Added bounded scan, refresh, grid, canvas, and reverse-DNS resource use.
- OS optimization now passes adapter values as data, has a one-minute deadline, and returns nonzero privilege and partial-failure outcomes.
- Result files are replaced atomically; file-write diagnostics use stderr; `NO_COLOR` and `--no-color` are honored.
- The authoritative version is now `const version` in `main.go`, using the required `1.xx.NNN` representation.
- Compatible common packages are consumed from the shared sibling `toolkit` module.

### Added
- Added JSON schema v1 at `docs/json-schema-v1.json` and Windows version resources with Xato ownership metadata.

### Breaking
- JSON now includes `schema_version: 1`, explicit host objects, and numeric millisecond fields instead of serialized Go implementation types.
- A custom output destination placed before the target must use `--flag=FILE`, removing hostname/filename ambiguity.

## [1.2.6] - 2026-09-11

### Added
- **Multi-Adapter Simultaneous Sweeping (`-A`, `--all-interfaces`)**:
  - **Concurrent Multi-Subnet Sweeping**: Scans all active, non-loopback network adapters simultaneously in parallel goroutines (e.g. Physical Ethernet LAN, Wi-Fi, Docker bridge, WSL vEthernet, VPN tunnels), mapping entire multi-homed network topologies in a single sweep cycle.
  - **Interactive Multi-Adapter HTML Dashboard (`--html`)**:
    - **Tabbed & Stacked Views**: Modern navigation tabs (`[ All Interfaces (Stacked) ]`, `[ Ethernet 2 ]`, `[ vEthernet (WSL) ]`) with instant client-side switching.
    - **Adapter-Specific Controls**: Independent Link Health HUD banners, filter pills (*All, Active, Fast, Slow, Silent, Offline*), zoom controls, and live state change trackers per interface.
    - **Global Action Controls**: One-click "Copy All Active" clipboard export across all interfaces, CSV export, JSON export, and dynamic dark/light theme switching.
  - **Stacked Terminal Display (`--ascii` / default)**:
    - Renders individual Link Health HUD banners, ASCII matrices, and interface summaries for each active network, followed by a consolidated multi-adapter summary.
  - **Grouped Host Inventory (`-l, --list`)**:
    - Generates interface-segmented columnar tables with resolved reverse DNS hostnames, IP addresses, MAC addresses, hardware manufacturers, and topology role badges.
  - **Structured Multi-Interface JSON (`--json`)**:
    - Outputs structured payloads containing top-level aggregate statistics (`total_hosts`, `total_active`, `interfaces_count`) alongside detailed per-interface arrays with link health and host telemetry.
  - **Multi-Grid PNG Composition (`--png`)**:
    - Stacks activity matrices vertically onto a unified high-resolution PNG image with canvas dividers and frame margins.
  - **CLI Flags & Shorthands**: Supported via `-A`, `--all-interfaces`, and `--all`.

---

## [1.2.5] - 2026-09-11

### Added
- **Interface-Aware System Optimizations (`pg optimize-os`)**:
  - **NIC-Level Energy Efficient Ethernet (EEE / 802.3az) Inspection & Control**: Inspects physical network interface hardware for EEE Low Power Idle (LPI) sleep states and provides automated remediation (`Set-NetAdapterAdvancedProperty` on Windows, `ethtool --set-eee <iface> eee off` on Linux) to prevent transceiver wake-up latency jitter and first-packet drop during fast subnet sweeps.
  - **Receive Side Scaling (RSS) Hardware Distribution**: Verifies both global TCP stack multi-core receive processing (`netsh int tcp show global`) and adapter-level RSS queues across all active NICs, enabling multi-core packet distribution to prevent single-core bottlenecking during high-concurrency scans.
  - **Interrupt Moderation Tuning**: Detects NIC interrupt moderation and coalescing configurations (Windows adapter properties and Linux `ethtool -c`), setting them to adaptive moderation for microsecond latency responses without packet drop.
  - **PowerShell & Ethtool Integration**: Automates adapter-level queries and updates while supporting `--dry-run` preview mode and full idempotency verification (`Already set to ...`).

---

## [1.2.4] - 2026-09-11

### Added
- **Local ARP / Neighbor Cache Integration**:
  - **OS Neighbor / ARP Cache Query**: Native high-speed ARP cache queries via `GetIpNetTable` (Windows `iphlpapi.dll`) and `/proc/net/arp` / `arp -an` (Linux/macOS), pre-warming network discovery and resolving hardware MAC addresses.
  - **Hardware Manufacturer OUI Lookups**: Built-in 24-bit OUI lookup engine mapping MAC address prefixes to hardware manufacturers (Apple, Cisco, Microsoft, Dell, Intel, HP, Lenovo, Raspberry Pi, Ubiquiti, TP-Link, Netgear, MikroTik, Espressif, Amazon, Google, Samsung, Sony, etc.).
  - **Silent / ICMP-Blocking Host Detection**: Automatically correlates ARP cache entries with ping sweep results; devices that respond to Layer 2 ARP requests but block ICMP Echo pings (such as Windows Defender Firewall, macOS, and IoT devices) are identified as `StatusSilent`.
  - **Visual Silent Host Markers**: Displayed with `?` in plain ASCII, bold amber `?` in ANSI color mode, separate `[?] Silent/Firewalled: N` count in ASCII legend, `.cell-silent` styling with dashed amber glow in HTML dashboards, and `silent` status with `- (arp)` RTT in tabular listings.
  - **Hardware Telemetry in Outputs**: MAC address and Vendor manufacturer columns added to tabular host listings (`-l`, `-l -v`), included in machine-readable JSON output (`--json`), and embedded in interactive HTML tooltips and HUD details.
  - **CLI Flags (`--arp` / `--no-arp`)**: Enabled by default; can be toggled or bypassed using `--no-arp`.

---

## [1.2.3] - 2026-09-11

### Added
- **Interface & Link Health HUD**: Contextual link health banner and widget displaying hardware description, physical link speed, duplex, MTU, DHCP lease status, and wireless correlation:
  - **Adapter Model & Hardware Description**: Identifies physical controller models (e.g. `Intel(R) Wi-Fi 6 AX201 160MHz`, `Microsoft Hyper-V Network Adapter #2`, `Realtek PCIe GbE Family Controller`).
  - **Link Speed & Duplex**: Evaluates negotiated link speed and duplex status (e.g. `1 Gbps Full Duplex`, `10 Gbps`, or wireless rates `1.2 Gbps`).
  - **MTU & DHCP Lease Status**: Displays interface MTU and dynamic DHCP lease state with remaining countdown (e.g. `Active (expires in 23h 14m)` or `Static IP`).
  - **Wi-Fi Correlation**: When connected to 802.11 wireless networks, reports current SSID, protocol standard (`Wi-Fi 6 (802.11ax)`), frequency band (`5 GHz`, `2.4 GHz`, `6 GHz`), channel number, and signal strength percentage with calibrated RSSI dBm (`94% (-53 dBm)`).
  - **Terminal Context Banner (`--hud` / `--no-hud`)**: Rendered directly above visual ASCII grids on local subnet sweeps.
  - **Interactive HTML Dashboard Widget**: A responsive, glassmorphic status card positioned above the subnet grid with live indicator badges.
  - **Enhanced Interface Inventory (`-I`, `--interfaces`)**: Added `SPEED` column to table mode and structured `link_health` object to machine-readable JSON payloads.

---

## [1.2.2] - 2026-09-11

### Added
- **Verbose Topology & Important Host Marking (`+`)**:
  - **Pre-Sweep Topology Discovery**: When verbose mode (`-v`, `--verbose`) is enabled, PingGrid prints all discovered local network topology hosts (Local Host / Me, Default Gateway, DNS Resolvers, DHCP Server) prefixed with `+` and their role badge before commencing the sweep loop.
  - **Live Sweep Marking**: During the sweep in verbose mode, important hosts (those with topological roles such as Me, Gateway, DNS, DHCP) are explicitly prefixed with `+` and their role badge.
  - **Offline Important Host Notification**: If an important topology host does not respond during the sweep, verbose mode explicitly logs `+ Host <IP> <Roles> did not respond (offline)`, ensuring critical infrastructure reachability issues are immediately visible while regular offline hosts remain suppressed.
  - **Tabular Host List Marking (`-l -v`)**: When rendering host lists with roles present, rows corresponding to important topology hosts are prefixed with `+ ` for immediate visual scanning.

---

## [1.2.1] - 2026-09-11

### Added
- **Multi-Role Topology Highlighting**: Automatically discovers and highlights key infrastructure and endpoint roles across the scanned network:
  - **Local Host ("Me")**: The local machine's IP on the swept interface is distinctly rendered with an `@` glyph in plain ASCII, bold bright cyan `\x1b[96;1m@\x1b[0m` in ANSI color mode, `[Me]` badge in sorted host lists, and a glowing cyan accent border in the interactive HTML dashboard.
  - **Default Gateway**: Detected primary router/gateway addresses maintain highlight styling with `^` and `[Gateway]` role badge.
  - **DNS Resolvers & DHCP Server**: Automatically queries configured DNS servers and DHCP server IP from adapter metadata, attaching `[DNS]` and `[DHCP]` badges to matching hosts in host list outputs and HTML dashboard hover tooltips.
  - **Adaptive ASCII Matrix Legend**: Dynamically displays `[@] Me: 1` alongside gateway and status counts when the local host is present in the swept range.

---

## [1.2.0] - 2026-09-11

### Added
- **Zero-Config Local Subnet Sweeping**: Running `pg` without positional target arguments now automatically sweeps the active primary local subnet instead of displaying the help screen. Help remains accessible via `pg -h`, `pg --help`, `pg /?`, or `pg /h`.
- **True Subnet Mask & Primary Gateway Detection**: Subnet auto-discovery detects the actual configured CIDR mask (e.g. `/22`, `/23`, `/25`, `/28`) from the active network interface rather than forcing a generic `/24`. Overly broad corporate masks (`/8`, `/16`) are safely constrained to a local `/24` block during zero-config execution to prevent runaway sweeps. Real default gateways are queried from kernel routing tables via `iphlpapi` / routing inspection.
- **Interface Targeting (`-i`, `--interface <name|index>`)**: Added `-i` / `--interface` flag to target and sweep the subnet of a specific network adapter by name or numeric index (e.g. `pg -i "Wi-Fi"`, `pg -i eth0`, `pg -i 1`). Supports all switch prefix variants (`/i`, `-i`, `--interface`, `/interface`).
- **Network Interface Inventory (`-I`, `--interfaces`)**: Added `-I` / `--interfaces` flag to inspect and list all local network adapters with their index, interface name, operational status, IPv4 address/CIDR, MAC hardware address, default gateway, and primary egress indicator. Supports machine-readable output with `--json`.

---

## [1.1.0] - 2026-09-10

### Added
- **Cross-Platform OS Network Stack Optimization (`--optimize-os` / `pg optimize-os`)**: Built-in automated kernel network optimizer for Windows, Linux, and macOS to maximize ICMP sweep throughput and neighbor cache performance.
  - **Windows**: Expands IPv4 global neighbor cache limit from 256 to 4096 (`netsh interface ipv4 set global neighborcachelimit=4096`), tunes interface base reachable duration to 5 minutes (`basereachable=300000`), reduces retransmission delay to 200 ms (`retransmit=200`), and creates a Windows Firewall outbound ICMP fastpath rule.
  - **Linux**: Tunes sysctl parameters (`mcast_solicit=1`, `retrans_time_ms=100`, `base_reachable_time_ms=300000`, `gc_thresh3=4096`, and enables `net.ipv4.ping_group_range="0 2147483647"` for unprivileged ICMP ping sockets).
  - **macOS**: Configures ARP cache longevity (`net.link.ether.inet.max_age=1200`), prune intervals, and expands socket buffer limits (`kern.ipc.maxsockbuf=4194304`).
  - **Elevation Safety & Dry Run**: Automatically detects elevation status (`IsUserAnAdmin` on Windows, UID 0 on POSIX), displaying detailed before/after values and actionable elevation instructions if non-elevated. Supports `--dry-run` inspection mode.

---

## [1.0.0] - 2026-09-10
 
### Added
- **Production 1.0.0 Release**: First official production release of PingGrid.
- **Embedded Visual Documentation Gallery**: Fully documented all 5 built-in color schemes (`dark`, `light`, `earth`, `moss`, `linen`), terminal ANSI matrix, plain monochrome ASCII mode, and interactive HTML dashboards with embedded high-resolution screenshots.

### Changed
- Concurrency worker pool default increased from 128 to 256, enabling simultaneous ping sweeps across entire `/24` subnets in a single concurrent pass.
- All latency metrics standardized to millisecond (`ms`) representation across console, HTML dashboards, and JSON outputs.

### Removed
- **Experimental ARP Cache Mode (`--arp-cache`)**: Removed the experimental `--arp-cache` flag and two-phase ARP logic. Parallel concurrent ICMP sweeping across the subnet is faster (~150ms total) than sequential ARP phase batching, provides accurate real-time latency across all nodes, and eliminates stale neighbor table blind spots.

---

## [0.6.0] - 2026-09-09

### Added
- **Broadcast Address Ping Prevention**: Ping sweeps automatically detect and skip transmitting ICMP echo requests to directed subnet broadcast addresses (such as `x.x.x.255` on `/24` subnets or calculated prefix broadcast addresses) and the global limited broadcast address `255.255.255.255`. Cells remain represented in their respective grid locations with offline status without generating network broadcast packets.
- **Universal CLI Switch Prefix Support (`-`, `--`, `/`)**: Parameters and flags can now be specified using single hyphens, double hyphens, or Windows-style slashes interchangeably (e.g. `/html`, `-html`, `--html`, `/p 3`, `-p 3`, `--p 3`, `/rows 5`, `-rows 5`, `--rows 5`, `/s moss`, `-scheme earth`, `/p:3`, `/rows=5`).
- **Millisecond Measurement Standard**: All latency and duration metrics across HTML dashboards, tooltips, delta notices, and verbose diagnostic logs are formatted in milliseconds (`ms`), removing microsecond (`µs`) display.

- **Shell Autocompletion for Bash, Zsh, and Fish**: Added dedicated completion subcommands (`pg completion bash`, `pg completion zsh`, `pg completion fish`) alongside `pg completion powershell`, including contextual completion for `--scheme`, `--color`, and `--encoding` flags.
- **Local ARP Neighbor Cache Inspection (`--arp-cache`)**: Added instant passive local ARP cache inspection mode via `--arp-cache`. Bypasses active ICMP ping transmission and reads the OS neighbor table in memory (<2ms, zero packets sent), clearly labeling output with `[ARP Cache Mode]`.

### Changed
- **Unrestricted Custom Color Hex Inputs**: Removed internal `AllowedPaletteHex` restrictions; users can supply any valid 3- or 6-digit hex color format (`#RRGGBB`, `RRGGBB`, `#RGB`, `RGB`, `0xRRGGBB`) to all `--color-*` override flags.
- **ASCII Plain Mode Slow Host Glyph (`*`)**: Slow-responding hosts in ASCII plain mode now use `*` instead of `!`. Fast / gateway infrastructure hosts use `^` to maintain clear visual distinction.
- **Moss Color Scheme Cell Background**: Updated the offline / no-response cell background color in the `moss` scheme to Forest Green (`#465a47`).
- **Earth Color Scheme Cell Background**: Updated the offline / no-response cell background color in the `earth` scheme to Warm Brown (`#886d5b`).
- **Linen Color Scheme Background & Foreground**: Configured `linen` scheme to use Warm Linen (`#f4eeeb`) for the cell background and Linen Taupe (`#897e79`) for active host foreground rendering.
- **Centralized Binary Build Output (`bin/`)**: Standardized all build workflows (`build.ps1` and `Makefile`) to output compiled executables directly into the `bin/` directory (`bin/pg.exe` on Windows, `bin/pg-linux-amd64` on Linux, and `bin/pg-darwin-*` on macOS), ensuring the root workspace directory remains clean.

### Removed
- **PowerShell Shortcut Flag (`--ps`)**: Removed the `--ps` flag in favor of `--json`. Standard `--json` produces clean, unadorned JSON summary output that integrates directly with PowerShell's `ConvertFrom-Json` without needing a dedicated shortcut mode.

---

## [0.5.0] - 2026-09-09

### Added
- **Dedicated Examples Option (`--examples`)**: Added `--examples` option displaying an organized reference of all supported IP target range formats (single IP, CIDR prefix, shorthand octet ranges, full IP ranges, wildcard patterns, bracket notation, explicit netmasks, comma-separated lists, and local subnet auto-detection) along with output format, scan tuning, and palette customization examples.
- **Network-Aware Timeout Scaling**: Ping timeouts automatically scale to the target network architecture. RFC1918 private subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), loopback (`127.0.0.0/8`), and link-local ranges default to a fast **150ms** timeout, while public/WAN destinations default to **400ms** (custom `--timeout` flags remain respected).
- **Concurrent Multi-Ping Execution**: When performing multiple pings (`-p`, default: 3), Ping 1 runs with the full timeout ($T$) while subsequent pings (Pings 2 & 3) execute with half timeout ($T/2$). All pings are dispatched concurrently without waiting for Ping 1 to finish, collecting replies and selecting the lowest round-trip latency.
- **Sorted Host List Format (`-l, --list [file]`)**: Added `-l, --list [file]` output format that produces a clean, columnar list of responding hosts sorted by ping time (RTT) ascending with automatic reverse DNS hostname resolution (optionally writing plain text to file or ANSI-highlighted to console).
- **Multi-Ping Count Option (`-p, --pings`)**: Added `-p, --pings <n>` allowing multiple ping attempts per host, recording the best/lowest round-trip time upon success.
- **Filename-Accepting Output Options**: Output options (`--html [file]`, `--iframe [file]`, `--png [file]`, `--json [file]`, `--summary [file]`, `--text [file]`) now directly accept an optional output filename, eliminating the need for a separate file flag.
- **Grouped CLI Help Structure**: Organized CLI `--help` into clean, dedicated functional sections with the most common group (**Output Options**) placed first, followed by **Scan Options**, **Grid Layout Options**, **Color & Styling Options**, and **Standard Options**.
- **Intelligent Grid Auto-Layout**: Rows and columns automatically size to fit the exact host count of the target IP range (e.g. 50 hosts -> 5&times;10, 100 hosts -> 10&times;10, 16 hosts -> 4&times;4, 64 hosts -> 4&times;16, 256 hosts -> 8&times;32), eliminating ghost/empty rows.
- **Adaptive ASCII Column Headers**: Header number indicators automatically adjust frequency based on column count (e.g. 1-to-1 indexing `0 1 2 ... 9` for 10-column layouts).
- **Parent Directory Creation in PNG Output**: `SavePNG` now automatically creates parent output directories if they do not already exist.

### Changed
- **Renamed `--text` to `--ascii`**: Renamed `--text` option to `--ascii` for terminal ASCII matrix output (optionally saving to file).
- **Reassigned `-v` to `--verbose`**: Made `-v` the standard shorthand for `--verbose`. Removed `-v` shorthand from version to prevent collision, adding `--ver` alongside `--version`.
- **Descriptive CLI Help Placeholders**: Replaced raw data types (`int`, `duration`, `string`) in help usage with clear descriptive placeholders (e.g. `<count>`, `<interval>`, `<duration>`, `<pixels>`, `<name>`, `<color>`).
- **Flexible Hex Color Parsing**: Hex color flags (`--color-online`, `--color-offline`, etc.) now accept values with or without the leading `#` (e.g. `4d86a2` as well as `#4d86a2`, `0x4d86a2`, and quoted strings), avoiding PowerShell comment truncation issues.
- **Removed `--gateway`**: Removed `--gateway` option; gateway highlighting is automatically assigned to the detected local gateway when sweeping the active subnet without explicit targets.
- **Removed `--fast-threshold`**: Removed `--fast-threshold` option. Responding hosts are classified as online or slow based on `--slow-threshold`, with highlight styling reserved for gateway/infrastructure addresses.
- **Removed `--yes` and `--no-input`**: Completely removed `--yes` and `--no-input` flags as PingGrid is fully non-interactive and does not contain user prompts.
- **Removed `--redact`**: Removed redundant `--redact` flag from CLI options as PingGrid does not process credentials or sensitive tokens.
- **Removed `-f, --output-file`**: Eliminated `-f` in favor of passing destination paths directly to output options (e.g. `--html dashboard.html`).
- **Removed `--output` (`-o`)**: Completely removed generic `--output` option in favor of explicit format options.
- **Removed `-t, --target`**: Eliminated redundant `--target` flag; targets are passed cleanly as positional arguments (e.g. `pg 192.168.1.0/24`).
- **Strict Host Determination from IP Range**: Target address ranges strictly define the scan size. Single IP targets (e.g. `192.168.1.1`) scan exactly 1 host (1&times;1 grid) rather than expanding into 256 sequential addresses.
- **Grid Capacity Error Enforcement**: If user specifies `-r` and `-c` dimensions smaller than the target host count (`len(ips) > rows * cols`), PingGrid immediately halts with a `GRID_CAPACITY_EXCEEDED` error instead of silently truncating hosts.
- **Uniform Constant Cell Sizing**: When rows and columns are autosized or customized, canvas width and height dynamically scale so cell sizes remain constant at 8&times;8 px in both PNG and HTML.
- **Theme Palette Refinements**: Refined built-in schemes: linen uses Warm Linen (`#f4eeeb`) for offline cells, earth uses Sage (`#b1b9a0`) online with Terracotta (`#ab7550`) highlight, and moss features Forest Moss (`#404e41`) offline with Sage (`#b1b9a0`) online.
- **BSD 2-Clause Licensing**: Dedicated the project under the open source BSD 2-Clause License.

---

## [0.4.0] - 2026-09-08

### Fixed
- **`-c` and `-r` Subnet Range Expansion**: Fixed an issue where `iprange.ParseAndExpand` used grid slot count as the safety limit, causing `INPUT_TOO_LARGE` errors when scanning subnets larger than the grid. Target parsing now uses `DefaultTargetLimit` and properly maps hosts to grid slots.
- **Empty Cell Rendering for Unmatched Range & Grid Sizes**: When range and grid sizes do not match (e.g. target has fewer IPs than grid capacity), extra cells are rendered with no color at all (transparent in PNG and HTML, blank two-space in ASCII console). Offline host metrics and legends now strictly report actual scanned hosts rather than inflating with empty grid slots.

### Changed
- **Proportional Grid Autosizing**: When `-c` (`--cols`) and/or `-r` (`--rows`) are specified without explicit width/height, the canvas dimensions autosize automatically (`AutosizeDimensions`) to maintain the standard 8x8 square cell size and frame padding. If only one dimension is specified, the other automatically scales to fit the address range.
- **Enforced Square Cells**: Both PNG image and HTML dashboard renderers now enforce square cell aspect ratios (`cellW == cellH`) across custom widths and heights.
- **Open Source Licensing**: Dedicated the project under a permissive open source license in `LICENSE`.

---

## [0.3.0] - 2026-09-04

### Added
- **Self-Contained Architecture**: Internalized all common toolkit dependencies (`cli`, `errors`, `log`, `paths`, `exec`, `iprange`) into `internal/toolkit/`, completely decoupling PingGrid from external sibling repositories.
- **Cross-Platform Build Script (`build.ps1`)**: Native PowerShell build script supporting Windows, Linux, and macOS builds with automatic metadata injection.
- **Cross-Compilation Targets**: Added `make build-linux`, `make build-darwin`, and `make build-all` to `Makefile`.
- **Supply-Chain & Vulnerability Pipeline**: Added `make verify-deps`, `make vuln`, `make ci`, and PowerShell `.\build.ps1 -Verify` integrating `go mod tidy`, `go mod download`, `go mod verify`, `go test`, `golangci-lint`, and `govulncheck`.
- **Border Alignment Unit Test**: Added `TestASCIIBorderAlignment` in `internal/grid/ascii_test.go` ensuring exact terminal column alignment.

### Changed
- **Binary Size Optimization**: Enabled `-trimpath` and `-ldflags="-s -w"` across `Makefile` and `build.ps1`, reducing executable footprint from 9.15 MB down to 6.32 MB (~31% size reduction).
- **Console Grid Alignment**: Adjusted leading spaces on top and bottom `+-----` border lines from 5 spaces to 6 spaces, aligning opening `+` and closing `+` with row `|` vertical borders.
- **Go Module Decoupling**: Removed `replace github.com/m8urnett/toolkit => ../toolkit` from `go.mod`.

---

## [0.2.0] - 2026-09-04

### Added
- **Dedicated Output Flags**: Replaced mandatory `-o` with dedicated first-class options: `--text`, `--json`, `--summary`, `--html`, and `--iframe` (alias: `--embed`).
- **Interactive Action Buttons in HTML Dashboard**: Added Refresh (`⟳ Refresh`), Copy Active IPs (`📋 Copy Active`), JSON download (`⬇ JSON`), CSV download (`⬇ CSV`), and Dark/Light Theme toggle (`🌓 Theme`).
- **Minimal Embeddable HTML (`--iframe`)**: Button-free, seamless transparent HTML page optimized strictly for iframe containers in dashboards and wikis.
- **Live State Delta Tracking**: Continuous monitoring mode (`-R`) tracks cycle-over-cycle host state transitions (`+` joined, `-` dropped, `~` changed) with console and HTML badges.
- **Positional Target Syntax**: Target subnet/IP can be provided optionless (e.g. `pg 10.8.0.1/24`).
- **Standard Help Switches**: Added Windows and Unix help switch aliases (`/?`, `-?`, `/h`, `/help`, `-h`, `--help`, `help`).
- **PowerShell Tab Autocompletion**: Integrated completion script generation under `pg completion powershell`.

### Changed
- **Binary Renaming**: Renamed binary target from `PingGrid.exe` to `pg.exe`.
- **Non-Destructive Defaults**: Default `-f` is now empty, ensuring console scans do not generate or overwrite disk files unless explicitly requested.
- **Conflict Validation**: Conflicting format flags (e.g. `--json --html`) raise `USAGE_CONFLICTING_FLAGS` error.

---

## [0.1.0] - 2026-09-04

### Added
- Initial release of PingGrid.
- High-concurrency IPv4 subnet ping sweeper using unprivileged Windows `iphlpapi.dll` and POSIX datagram sockets.
- Curated 9-color palette with preset themes (`dark`, `light`, `earth`, `moss`, `linen`).
- Terminal ASCII activity matrix with 24-bit TrueColor ANSI and plain monochrome fallbacks.
- PNG raster image export matching the reference layout (295x77px, 8x32 cells).
