package cmd

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [file] [directory] ...",
	Short: "Apply the supplied stack configuration",
	Long:  `Apply the supplied stack configuration`,
	Example: `  # Recursively create a stack from a directory
  yard apply my-stack

  # Create a stack from a specific file
  yard apply my-stack/network.hcl
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}