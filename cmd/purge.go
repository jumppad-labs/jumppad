package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

		// Remove any images whcih have been built
		filter := filters.NewArgs()
		filter.Add("reference", "shipyard.run/localcache/*")

		// check if the image already exists, if so do not rebuild unless force
		sum, err := dt.ImageList(context.Background(), types.ImageListOptions{Filters: filter})
		if err != nil {
			return fmt.Errorf("Unable to check image cache, error: %s", err)
		}

		for _, i := range sum {
			l.Info("Removing image", "image", i.ID)

			_, err := dt.ImageRemove(context.Background(), i.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				return fmt.Errorf("Unable to delete image: %s, error: %s", i.ID, err)
			}
		}

		l.Info("Removing cached images for clusters")
		err = dt.VolumeRemove(context.Background(), utils.FQDNVolumeName("images"), true)
		if err != nil {
			return fmt.Errorf("Unable to remove cached image volume, error: %s", err)
		}

		hcp := utils.GetBlueprintLocalFolder("")
		l.Info("Removing Blueprints", "path", hcp)
		err = os.RemoveAll(hcp)
		if err != nil {
			return fmt.Errorf("Unable to remove cached Helm charts: %s", err)
		}

		bcp := utils.GetHelmLocalFolder("")
		l.Info("Removing Helm charts", "path", bcp)
		err = os.RemoveAll(bcp)
		if err != nil {
			return fmt.Errorf("Unable to remove cached Blueprints: %s", err)
		}

		// delete the releases
		rcp := utils.GetReleasesFolder()
		l.Info("Removing releases", "path", rcp)
		err = os.RemoveAll(rcp)
		if err != nil {
			return fmt.Errorf("Unable to remove cached Releases: %s", err)
		}

		dcp := utils.GetDataFolder("")
		l.Info("Removing data folder", "path", dcp)
		err = os.RemoveAll(dcp)
		if err != nil {
			return fmt.Errorf("Unable to remove Data folder: %s", err)
		}

		return nil
	}
}
