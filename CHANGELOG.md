# Changelog

All notable changes to PingGrid will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-09-04

### Added
- **Self-Contained Architecture**: Internalized all common toolkit dependencies (`cli`, `errors`, `log`, `paths`, `exec`, `iprange`) into `internal/toolkit/`, completely decoupling PingGrid from external sibling repositories.
- **Cross-Platform Build Script (`build.ps1`)**: Native PowerShell build script supporting Windows, Linux, and macOS builds with automatic metadata injection.
- **Cross-Compilation Targets**: Added `make build-linux`, `make build-darwin`, and `make build-all` to `Makefile`.
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
