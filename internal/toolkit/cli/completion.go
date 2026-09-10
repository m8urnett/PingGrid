package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// GenerateBashCompletion outputs a Bash completion script to stdout.
func GenerateBashCompletion(cmd *cobra.Command) error {
	return cmd.GenBashCompletion(os.Stdout)
}

// GenerateZshCompletion outputs a Zsh completion script to stdout.
func GenerateZshCompletion(cmd *cobra.Command) error {
	return cmd.GenZshCompletion(os.Stdout)
}

// GenerateFishCompletion outputs a Fish completion script to stdout.
func GenerateFishCompletion(cmd *cobra.Command) error {
	return cmd.GenFishCompletion(os.Stdout, true)
}
