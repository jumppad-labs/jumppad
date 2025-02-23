package cmd

import (
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:           "version",
		Short:         "jumppad version manager commands",
		Long:          "jumppad version manager commands",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Current Version:", version)
			cmd.Println("")

			return nil
		},
	}

	return versionCmd
}
