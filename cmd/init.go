package cmd

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use: "init [directory]",
	Short: "Create a new stack",
	Long:  `Create a new stack`,
	Example: `yard init my-stack`,
	DisableFlagsInUseLine: true,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}