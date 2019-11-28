package cmd

import (
	"github.com/spf13/cobra"
)

var docsPort int

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Show the documentation for the current stack",
	Long:  `Run the docs container and exposes the documentation for the current stack on the defined port (default is 8080)`,
	DisableFlagsInUseLine: true,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func init() {
	docsCmd.PersistentFlags().IntVarP(&docsPort, "port", "p", 8080, "the port to expose the docs on")
}