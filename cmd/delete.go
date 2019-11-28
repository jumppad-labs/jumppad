package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [file] [directory] ...",
	Short: "Delete the current stack",
	Long:  `Delete the current stack`,
	Example: `yard delete my-stack`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}