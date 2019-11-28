package cmd

import (
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Show the the editor for current stack",
	Long:  `Run the vscode container and exposes the editor for the current stack on the defined port (default is 9000)`,
	DisableFlagsInUseLine: true,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}