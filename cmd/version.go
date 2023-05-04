package cmd

import (
	"fmt"

	gvm "github.com/shipyard-run/version-manager"
	"github.com/spf13/cobra"
)

func newVersionCmd(vm gvm.Versions) *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:           "version",
		Short:         "jumppad version manager commands",
		Long:          "jumppad version manager commands",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Current Version:", version)
			cmd.Println("")

			return fmt.Errorf("")
		},
	}

	versionCmd.AddCommand(newVersionListCmd(vm))
	versionCmd.AddCommand(newVersionInstallCmd(vm))
	return versionCmd
}
