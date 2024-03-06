package cmd

import (
	"os"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newDestroyCmd(cc connector.Connector, l logger.Logger) *cobra.Command {
	var force bool

	downCmd := &cobra.Command{
		Use:     "down",
		Short:   "Remove all resources in the current state",
		Long:    "Remove all resources in the current state",
		Example: `jumppad down`,
		Run: func(cmd *cobra.Command, args []string) {
			engineClients, _ := clients.GenerateClients(l)
			engineClients.ContainerTasks.SetForce(force)

			engine, _, err := createEngine(l, engineClients)
			if err != nil {
				l.Error("Unable to create engine", "error", err)
				return
			}

			logger := createLogger()

			logger.Debug("Destroying stack", "force", force)

			err = engine.Destroy(force)
			if err != nil {
				l.Error("Unable to destroy stack", "error", err)
				return
			}

			// clean up the data folders
			os.RemoveAll(utils.DataFolder("", os.ModePerm))
			os.RemoveAll(utils.LibraryFolder("", os.ModePerm))

			// shutdown ingress when we destroy all resources
			if cc.IsRunning() {
				err = cc.Stop()
				if err != nil {
					logger.Error("Unable to destroy jumppad daemon", "error", err)
				}
			}
		},
	}

	downCmd.Flags().BoolVarP(&force, "force", "", false, "When set to true Jumppad will not wait for containers to exit gracefully and will ignore errors")

	return downCmd
}
