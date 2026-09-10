# Changelog

All notable changes to PingGrid will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
