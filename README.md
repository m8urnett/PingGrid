# PingGrid

PingGrid is a high-performance, cross-platform network ping sweeper and subnet visualizer for IPv4 address ranges. It concurrently pings target addresses and renders an activity grid in multiple formats: **console ASCII/ANSI matrix**, **interactive HTML dashboard**, and **PNG image**.

Runs natively on **Windows**, **Linux**, and **macOS**.

## Quick Start

```sh
# Sweep target subnet (positional target syntax)
pg.exe 192.168.1.0/24

# Continuous live monitoring: refresh sweep every 5 seconds (auto-refreshes terminal & HTML)
pg.exe 192.168.1.0/24 -R 5s

# Standalone interactive HTML dashboard (filters, clipboard copy, zoom, HUD)
pg.exe 192.168.1.0/24 --html -f dashboard.html

# Minimal embeddable HTML for iframes (zero buttons, transparent, border-fit)
pg.exe 192.168.1.0/24 --iframe -f embed.html

# Plain monochrome ASCII mode for simple terminals or automation
pg.exe 192.168.1.0/24 --plain

# Output as PowerShell-friendly JSON summary
pg.exe 192.168.1.0/24 --ps

# Display help and CLI usage (pg.exe without parameters also displays help)
pg.exe /?
```

## Grid Layout & Color Representation

By default, PingGrid generates an activity grid matching the layout of `grid.png` (**295 &times; 77 px**, **8 rows &times; 32 columns** = 256 cells, representing a standard `/24` subnet).

### Color Palette & Meanings

PingGrid uses a curated 9-color palette: `#2c2c2c`, `#efefef`, `#f4eeeb`, `#b1b9a0`, `#4d86a2`, `#ab7550`, `#404e41`, `#fcfcfc`, and `#e4f9d4`.

The default scheme (`--scheme dark`) maps these colors to:

| State / Role | Hex Code | Character | Meaning |
|---|---|---|---|
| **Outer Frame & Dividers** | `#2c2c2c` | `+ - \|` | Canvas bezel, borders, and column dividers. |
| **Offline / Unresponsive** | `#404e41` | `·` | Scanned IP did not reply within timeout. |
| **Active / Online** | `#4d86a2` | `o` | Host responded normally (< `--slow-threshold`). |
| **Slow / Degraded** | `#ab7550` | `!` | High latency node (&ge; `--slow-threshold`, default 100ms). |
| **Fast / Highlight** | `#e4f9d4` | `*` | Ultra-low latency (< `--fast-threshold`, default 20ms) or gateway IP. |

### Built-in Schemes

Use `--scheme` / `-s` to switch between preset color combinations:

- **`dark`** (default): Charcoal frame/border (`#2c2c2c`), Moss offline (`#404e41`), Teal online (`#4d86a2`), Terracotta slow (`#ab7550`), Mint fast (`#e4f9d4`).
- **`light`**: Warm Linen frame (`#f4eeeb`), Light Gray border (`#efefef`), Pure Off-White offline (`#fcfcfc`), Teal online (`#4d86a2`), Charcoal slow (`#2c2c2c`), Terracotta fast (`#ab7550`).
- **`earth`**: Charcoal frame (`#2c2c2c`), Moss offline (`#404e41`), Sage online (`#b1b9a0`), Slate Teal slow (`#4d86a2`), Terracotta fast (`#ab7550`).
- **`moss`**: Moss frame (`#404e41`), Charcoal border/offline (`#2c2c2c`), Teal online (`#4d86a2`), Terracotta slow (`#ab7550`), Mint fast (`#e4f9d4`).
- **`linen`**: Warm Linen frame (`#f4eeeb`), Light Gray border (`#efefef`), Off-White offline (`#fcfcfc`), Moss online (`#404e41`), Terracotta slow (`#ab7550`), Teal fast (`#4d86a2`).

All dimensions, grid arrangements, and individual colors can also be overridden via CLI flags.

## Command-Line Options

```
Usage:
  pg [target] [flags]

Flags:
      --text                      Output console ASCII text grid (default)
      --json                      Output machine-readable JSON summary
      --summary                   Output single-line text summary
      --html                      Generate standalone interactive HTML dashboard (defaults to grid.html)
      --iframe                    Generate embeddable minimal HTML for iframes (defaults to grid-embed.html; alias: --embed)
  -o, --output string             Output format: text, json, summary, html, iframe (default "text")
  -s, --scheme string             Built-in color scheme: dark, light, earth, moss, linen (default "dark")
  -t, --target string             Target subnet CIDR (e.g. 192.168.1.0/24) or base IP (auto-detects local subnet if omitted)
  -R, --refresh duration          Continuous sweep refresh interval (e.g. 5s, 10s; 0 runs once)
  -r, --rows int                  Number of grid rows (default 8)
  -c, --cols int                  Number of grid columns (default 32)
  -W, --width int                 Output image width in pixels (default 295)
  -H, --height int                Output image height in pixels (default 77)
      --border-width int          Grid divider border width in pixels (default 1)
  -f, --output-file string        Path for output file (e.g. grid.html, embed.html, or .png)
      --color-offline string      Hex color for offline hosts (default "#404e41")
      --color-online string       Hex color for active hosts (default "#4d86a2")
      --color-highlight string    Hex color for fast/highlight hosts (default "#e4f9d4")
      --color-slow string         Hex color for slow-responding hosts (default "#ab7550")
      --color-border string       Hex color for cell divider lines (default "#2c2c2c")
      --color-frame string        Hex color for outer canvas frame (default "#2c2c2c")
      --concurrency int           Concurrent ping workers (default 128)
      --fast-threshold duration   Latency threshold for highlight color (default 20ms)
      --slow-threshold duration   Latency threshold for slow/degraded color (default 100ms)
      --gateway string            Optional gateway IP to always highlight
  -v, --version                   Display version information and exit

Standard Flags:
      --ps                        Enable PowerShell mode (implies --json --plain)
      --plain                     Disable ANSI formatting and colors (monochrome glyphs)
      --timeout duration          Per-host ping timeout (default 500ms)
      --quiet                     Suppress progress output
      --verbose                   Show per-host latency details with microsecond precision

Help & Usage:
  Running pg without arguments displays full usage.
  Standard Windows and Unix help switches are supported: /?, -?, /h, -h, --help, /help, help.
```

## Shell Completion

Generate shell autocompletion for `pg` commands, flags, built-in color schemes, and output formats:

```powershell
# Load completion in the current PowerShell session:
pg.exe completion powershell | Out-String | Invoke-Expression

# Persist completion across all PowerShell sessions ($PROFILE):
Add-Content $PROFILE "`npg.exe completion powershell | Out-String | Invoke-Expression"
```

## Examples

### Terminal Subnet Sweep (Default Text Mode)
```sh
# Sweep target subnet and display ASCII activity matrix to console (no files generated)
pg.exe 10.8.0.1/24
```

### Continuous Live Subnet Monitor
```sh
# Sweep local subnet every 3 seconds with state delta tracking
pg.exe -R 3s
```

### Standalone Interactive HTML Dashboard
```sh
# Generate standalone interactive HTML dashboard (defaults to grid.html)
pg.exe 10.8.0.1/24 --html

# Specify a custom destination filename:
pg.exe 10.8.0.1/24 --html -f dashboard.html
```

### Embeddable Minimal HTML for Iframes
```sh
# Generate minimal button-free HTML ready for embedding in dashboards or wikis (defaults to grid-embed.html)
pg.exe 10.8.0.1/24 --iframe -f embed.html
```

### Structured Output (JSON / Summary)
```sh
# Output machine-readable JSON summary for automation and pipelines:
pg.exe 10.8.0.1/24 --json

# Output single-line summary:
pg.exe 10.8.0.1/24 --summary
```

### Custom Grid Dimensions and Colors
```sh
# 16x16 grid (256 addresses) rendered as a 512x512 PNG with custom colors
pg.exe -t 10.0.0.0/24 -r 16 -c 16 -W 512 -H 512 --color-online "#4CAF50" -f lan-matrix.png
```

### Verbose Scan with Microsecond Resolution
```sh
pg.exe --target 10.10.1.0/24 --verbose --timeout 1s
```

## Output Formats: HTML, Embed, Text, & JSON

PingGrid provides dedicated flags to select the desired output format:

1. **Standalone Dashboard (`--html`, `-o html`)**:
   - Designed for full-screen browser viewing and interactive analysis.
   - **Header Action Buttons**:
     - **Refresh (`⟳ Refresh`)**: Instant manual reload trigger for the dashboard.
     - **Copy Active IPs (`📋 Copy Active (X)`)**: One-click clipboard export of responding host IPs with dynamic count and visual confirmation.
     - **Export JSON (`⬇ JSON`)**: Client-side export and download of complete host scan results as `pinggrid-results.json`.
     - **Export CSV (`⬇ CSV`)**: Client-side export and download of scan metrics (`IP,Status,RTT,Delta`) as `pinggrid-results.csv`.
     - **Theme Toggle (`🌓 Theme`)**: Instant client-side switching between sleek dark mode and light mode.
   - **Click-to-Filter Controls**: Filter the matrix by status (*All*, *Active*, *Fast*, *Slow*, *Offline*, *Deltas*) with dynamic cell dimming.
   - **Interactive HUD & Zoom**: Displays host IP, classification status, microsecond RTT, and 1x/2x/3x zoom scaling.
   - **Live State Delta Log**: Detailed collapsible tracking card highlighting newly joined (`+`), dropped (`-`), or latency-shifted (`~`) hosts across sweeps.
   - **Countdown & Pause**: Live countdown ticker with pause/resume button when running in continuous monitoring mode (`-R`).

2. **Minimal Embed / Iframe Page (`-o iframe`)**:
   - Designed strictly for embedding in `<iframe>` containers within dashboards or wikis (e.g. `<iframe src="embed.html"></iframe>`).
   - **Zero Buttons**: No zoom buttons, no pause buttons, no filter pills, and no action buttons.
   - **Seamless Fitting**: Transparent background and zero-padding frame wrapping only the pixel-exact activity canvas.
   - **Native Tooltips & Delta Animations**: Hover tooltips and keyframe animations (`joinPulse`, `dropBlink`) preserved.
   - **Silent Auto-Reload**: Automatically refreshes in sync with the sweep loop when `-R` is configured.

## Platform Architecture

- **Windows Native ICMP**: Utilizes `iphlpapi.dll`'s `IcmpSendEcho` API from user-mode without requiring Administrator / elevated privileges. High-resolution monotonic timers measure round-trip times down to microsecond accuracy.
- **Linux & macOS ICMP**:
  - **Unprivileged Datagram ICMP**: Employs datagram sockets (`udp4`), supported natively on macOS and Linux distros with `ping_group_range`.
  - **Privileged Raw Socket**: Uses raw ICMP sockets (`ip4:icmp`) when running as root or with `CAP_NET_RAW`.
  - **Subprocess Fallback**: Automatically invokes the system `ping` binary if raw/datagram sockets are restricted.
  - **TCP Discovery Probing**: Probes key ports (53, 80, 445, 22) if ICMP is blocked by firewall policies.
- **Cross-Platform Auto-Detection**: Detects primary active IPv4 network interface and computes the enclosing `/24` subnet and gateway automatically.

## Build & Development

### Prerequisites

1. **Go Toolchain**: Requires Go 1.23 or later.
2. **Self-Contained**: PingGrid is completely self-contained with zero private module dependencies. You can clone the repository to any directory and build immediately without requiring sibling folders or external tokens.

### Building on Windows (PowerShell)

Use the included native PowerShell build script:

```powershell
# Build pg.exe for Windows:
.\build.ps1

# Cross-compile for Linux (amd64):
.\build.ps1 -Target linux

# Build all platform binaries (Windows, Linux, macOS arm64/amd64):
.\build.ps1 -Target all
```

### Building with Make (Linux / macOS / GNU Make)

```sh
# Build default binary (pg.exe on Windows, pg on POSIX)
make build

# Cross-compile for Linux (amd64)
make build-linux

# Cross-compile for macOS (Apple Silicon + Intel)
make build-darwin

# Cross-compile all targets
make build-all

# Run full test suite
make test

# Format, vet, and verify
make check
```

### Direct Go CLI Build

```sh
# Standard build:
go build -o pg.exe .

# Linux static binary (pure Go, zero CGO):
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pg-linux-amd64 .
```

### Build & Compilation Notes

- **Version & Build Metadata Injection**: Builds inject the Git commit hash and UTC build timestamp into the executable at link time using `-ldflags`:
  ```sh
  go build -ldflags "-X github.com/m8urnett/PingGrid/internal/version.GitCommit=$(git rev-parse --short HEAD) -X github.com/m8urnett/PingGrid/internal/version.BuildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" -o pg.exe .
  ```
  The authoritative version and build number are defined in [`internal/version/version.go`](file:///c:/Users/mark/Proton%20Drive/m8urn/My%20files/(Dev)/PingGrid/internal/version/version.go).
- **Pure Go Static Binaries (`CGO_ENABLED=0`)**: Linux and macOS cross-compilation builds set `CGO_ENABLED=0` to create completely static, portable executables with zero external C library or glibc dependencies.
- **Linux Network Capabilities**: Windows uses `iphlpapi.dll` without requiring elevation. On Linux, if unprivileged ICMP datagram sockets (`udp4`) are disabled by default on your distribution, grant socket capabilities or enable the sysctl ping group range:
  ```sh
  # Option A: Grant raw socket capability to binary
  sudo setcap cap_net_raw+ep ./bin/pg-linux-amd64

  # Option B: Enable unprivileged ICMP sockets system-wide
  sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
  ```
- **Self-Contained / Offline Builds (`vendor`)**: In air-gapped or offline CI environments, you can vendor all third-party dependencies using:
  ```sh
  go mod vendor
  go build -mod=vendor -o pg.exe .
  ```
- **Automated Tests**: Run the full unit test suite covering color schemes, HTML templates, state deltas, CIDR generation, and CLI argument conflict validation:
  ```sh
  go test -v ./...
  ```

## License

See [LICENSE](file:///c:/Users/mark/Proton%20Drive/m8urn/My%20files/(Dev)/PingGrid/LICENSE).

