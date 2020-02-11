package cmd

import (
	"fmt"
	"os"

	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:     "destroy [file] [directory] ...",
	Short:   "Destroy the current stack",
	Long:    `Destroy the current stack`,
	Example: `yard destroy`,
	Run: func(cmd *cobra.Command, args []string) {

		log := createLogger()

		// When destroying a stack all the config
		// which is created with apply is copied
		// to the state folder
		e, err := shipyard.NewFromState(log)
		if err != nil {
			log.Error("Unable to load state", "error", err)
			return
		}

		fmt.Printf("Destroying %d resources\n\n", e.ResourceCount())

		err = e.Destroy()
		if err != nil {
			log.Error("Unable to destroy stack", "error", err)
			return
		}

		/*
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
		*/

		// delete the contents of the state folder
		err = os.RemoveAll(utils.StateDir())
		if err != nil {
			log.Error("Unable to delete state", "error", err)
			return
		}
	},
}
