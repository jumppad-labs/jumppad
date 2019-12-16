package cmd

import (
	"fmt"
	"os"

	"github.com/shipyard-run/cli/pkg/shipyard"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [file] [directory] ...",
	Short:   "Delete the current stack",
	Long:    `Delete the current stack`,
	Example: `yard delete my-stack`,
	Run: func(cmd *cobra.Command, args []string) {
		// When destroying a stack all the config
		// which is created with apply is copied
		// to the state folder
		e, err := shipyard.NewWithFolder(StateDir())
		if err != nil {
			panic(err)
		}

		err = e.Destroy()
		if err != nil {
			panic(err)
		}

		// remove the environment varibles
		if e.Blueprint() != nil && len(e.Blueprint().Environment) > 0 {
			fmt.Println("restoring environment variables")
			ef, err := NewEnv(fmt.Sprintf("%s/env.var", StateDir()))
			if err != nil {
				panic(err)
			}
			defer ef.Close()

			err = ef.Clear()
			if err != nil {
				panic(err)
			}
		}

		// delete the contents of the state folder
		err = os.RemoveAll(StateDir())
		if err != nil {
			panic(err)
		}
	},
}
