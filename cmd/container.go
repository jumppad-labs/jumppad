package cmd

import (
	"github.com/spf13/cobra"
)

var containerCmd = &cobra.Command{
	Use:   "container [subcommand]",
	Short: "The container subcommand...",
	Long:  `The container subcommand...`,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}