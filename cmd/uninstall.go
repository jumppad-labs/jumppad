package cmd

import (
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall shipyard",
	Long:  `Uninstall shipyard`,
	DisableFlagsInUseLine: true,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}