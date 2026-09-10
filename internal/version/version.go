package version

import "fmt"

// Authoritative version information for PingGrid.
const (
	Major = 0
	Minor = 6
	Patch = 0
	Build = 42
)

var (
	// Version is the canonical SemVer release string.
	Version = fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)

	// GitCommit is set at link time via -ldflags.
	GitCommit = "dev"

	// BuildDate is set at link time via -ldflags.
	BuildDate = "unknown"
)

// String returns the canonical version string.
func String() string {
	return Version
}

// Full returns detailed version and build information.
func Full() string {
	return fmt.Sprintf("pg %s (build %d, commit %s, built %s)", Version, Build, GitCommit, BuildDate)
}
