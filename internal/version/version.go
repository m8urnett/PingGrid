package version

import "fmt"

var (
	// GitCommit is set at link time via -ldflags.
	GitCommit = "dev"

	// BuildDate is set at link time via -ldflags.
	BuildDate = "unknown"
)

// Full returns detailed version and build information.
func Full(releaseVersion string) string {
	return fmt.Sprintf("pg %s (commit %s, built %s)", releaseVersion, GitCommit, BuildDate)
}
