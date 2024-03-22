package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

			credentials := map[string]string{}
			for k, v := range viper.GetStringMap("credentials") {
				credentials[k] = v.(string)
			}

			engine, err := createEngine(l, engineClients, credentials)
			if err != nil {
				l.Error("Unable to create engine", "error", err)
				return
			}

			logger := createLogger()

			done := make(chan os.Signal, 1)
			signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cmd.Println("Destroying resources", " -- press ctrl c to cancel")
			cmd.Println("")

			logger.Debug("Destroying stack, press ctrl-c to stop", "force", force)

			go func() {
				<-done // Will block here until user hits ctrl+c

				// cancel the context
				cancel()
			}()

			err = engine.Destroy(ctx, force)
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
