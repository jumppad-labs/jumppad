package cmd

import (
	gvm "github.com/shipyard-run/version-manager"
	"github.com/spf13/cobra"
)

var version string // set by build process
var commit string  // set by build process
var date string    // set by build process

func newVersionCmd(_ gvm.Versions) *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:           "version",
		Short:         "jumppad version manager commands",
		Long:          "jumppad version manager commands",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Version: %s\n", version)
			cmd.Printf("Commit:  %s\n", commit)
			cmd.Printf("Date:    %s\n", date)
			cmd.Println()

			return nil
		},
	}

	return versionCmd
}
