package app

import "context"

// Execute constructs and runs the PingGrid command tree for the supplied arguments.
func Execute(ctx context.Context, args []string, releaseVersion string) error {
	normalizedArgs := normalizeCLIArgs(args)
	rootCmd, _ := newRootCmdForVersion(releaseVersion)
	rootCmd.SetContext(ctx)
	if len(normalizedArgs) > 0 {
		rootCmd.SetArgs(normalizedArgs[1:])
	}
	return rootCmd.Execute()
}
