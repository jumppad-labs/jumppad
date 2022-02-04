package cmd

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

func newDestroyCmd(cc clients.Connector) *cobra.Command {
	return &cobra.Command{
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

			if dst == "" {
				// clean up the data folder
				os.RemoveAll(utils.GetDataFolder(""))

				// remove the certs
				os.RemoveAll(utils.CertsDir(""))
			}

			// shutdown ingress when we destroy all resources
			if cc.IsRunning() && dst == "" {
				err = cc.Stop()
				if err != nil {
					hclog.Default().Error("Unable to stop ingress", "error", err)
				}
			}
		},
	}
}
