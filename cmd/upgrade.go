package cmd

import (
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Shipyard",
	Long:  `Upgrade the Shipyard binary, but leaves the stacks alone`,
	DisableFlagsInUseLine: true,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}