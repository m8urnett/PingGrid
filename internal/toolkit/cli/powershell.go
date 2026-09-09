package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// GeneratePowerShellCompletion outputs a Register-ArgumentCompleter script.
func GeneratePowerShellCompletion(cmd *cobra.Command) error {
	return cmd.GenPowerShellCompletion(os.Stdout)
}
