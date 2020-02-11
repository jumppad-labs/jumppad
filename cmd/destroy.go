package cmd

import (
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [file]",
	Short: "Destroy the current stack or file",
	Long: `Destroy the current stack or file. 
	If the optional parameter "file" is passed then only the resources contained
	in the file will be destroyed`,
	Example: `yard destroy`,
	Run: func(cmd *cobra.Command, args []string) {
		log := createLogger()

		dst := ""
		if len(args) > 0 {
			dst = args[0]
		}

		// When destroying a stack all the config
		// which is created with apply is copied
		// to the state folder
		var e *shipyard.Engine
		var err error

		e, err = shipyard.New(log)
		if err != nil {
			log.Error("Unable to load file", "error", err)
			return
		}

		if dst == "" {
			err = e.Destroy(dst, true)
		} else {
			err = e.Destroy(dst, false)
		}

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
	},
}
