package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
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

			engine, err := createEngine(l, engineClients)
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

			// Clean up the data folder. Containers managed by Jumppad (for
			// example Vault) may have written files into a mounted data folder
			// using a UID that does not belong to the host user, so host side
			// removal will fail with EPERM. Run a short lived alpine container
			// to remove the folder as root.
			dataPath := filepath.Join(utils.JumppadHome(), "data")
			if _, err := os.Stat(dataPath); err == nil {
				if cerr := container.CleanupHostPath(engineClients.ContainerTasks, logger, dataPath); cerr != nil {
					logger.Error("Unable to clean data folder", "path", dataPath, "error", cerr)
				}
			}

			if err := os.RemoveAll(utils.LibraryFolder("", os.ModePerm)); err != nil {
				logger.Error("Unable to remove library folder", "error", err)
			}
			if err := os.RemoveAll(utils.JumppadTemp()); err != nil {
				logger.Error("Unable to remove temp folder", "error", err)
			}

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
