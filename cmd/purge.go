package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

func newPurgeCmd(dt clients.Docker, il clients.ImageLog, l hclog.Logger) *cobra.Command {
	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Purges Docker images, Helm charts, and Blueprints downloaded by Shipyard",
		Long:  "Purges Docker images, Helm charts, and Blueprints downloaded by Shipyard",
		Example: `
  shipyard purge
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newPurgeCmdFunc(dt, il, l),
		SilenceUsage: true,
	}

	return purgeCmd
}

func newPurgeCmdFunc(dt clients.Docker, il clients.ImageLog, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		images, _ := il.Read(clients.ImageTypeDocker)

		for _, i := range images {
			l.Info("Removing image", "image", i)

			_, err := dt.ImageRemove(context.Background(), i, types.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				return fmt.Errorf("Unable to delete image: %s, error: %s", i, err)
			}
		}
		il.Clear()

		hcp := filepath.Join(utils.ShipyardHome(), "helm_charts")
		l.Info("Removing Helm charts", "path", hcp)
		err := os.RemoveAll(hcp)
		if err != nil {
			return fmt.Errorf("Unable to remove cached Helm charts: %s", err)
		}

		bcp := filepath.Join(utils.ShipyardHome(), "helm_charts")
		l.Info("Removing Blueprints", "path", bcp)
		err = os.RemoveAll(bcp)
		if err != nil {
			return fmt.Errorf("Unable to remove cached Blueprints: %s", err)
		}

		return nil
	}
}
