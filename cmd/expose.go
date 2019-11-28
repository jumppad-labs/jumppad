package cmd

import (
	"github.com/spf13/cobra"
)

var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose a service in the stack on the host machine",
	Long:  `Expose a service in the stack on the host machine`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}