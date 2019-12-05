package cmd

import (
	"github.com/shipyard-run/cli/pkg/shipyard"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [file] [directory] ...",
	Short:   "Delete the current stack",
	Long:    `Delete the current stack`,
	Example: `yard delete my-stack`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		e, err := shipyard.NewWithFolder(args[0])
		if err != nil {
			panic(err)
		}

		err = e.Destroy()
		if err != nil {
			panic(err)
		}
	},
}
