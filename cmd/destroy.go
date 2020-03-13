package cmd

import (
	"github.com/hashicorp/go-hclog"
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
		dst := ""
		if len(args) > 0 {
			dst = args[0]
		}

		// When destroying a stack all the config
		// which is created with apply is copied
		// to the state folder
		var err error
		if dst == "" {
			err = engine.Destroy(dst, true)
		} else {
			err = engine.Destroy(dst, false)
		}

		if err != nil {
			hclog.Default().Error("Unable to destroy stack", "error", err)
			return
		}
	},
}
