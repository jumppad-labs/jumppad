package cmd

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newDestroyCmd(cc clients.Connector) *cobra.Command {
	return &cobra.Command{
		Use:     "down",
		Short:   "Remove all resources in the current state",
		Long:    "Remove all resources in the current state",
		Example: `jumppad down`,
		Run: func(cmd *cobra.Command, args []string) {
			err := engine.Destroy()
			if err != nil {
				hclog.Default().Error("Unable to destroy stack", "error", err)
				return
			}

			// clean up the data folder
			os.RemoveAll(utils.GetDataFolder("", os.ModePerm))

			// remove the certs
			os.RemoveAll(utils.CertsDir(""))

			// shutdown ingress when we destroy all resources
			if cc.IsRunning() {
				err = cc.Stop()
				if err != nil {
					hclog.Default().Error("Unable to stop ingress", "error", err)
				}
			}
		},
	}
}
