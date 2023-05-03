package cmd

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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

		bHasError := false

		for _, i := range images {
			l.Info("Removing image", "image", i)

			_, err := dt.ImageRemove(context.Background(), i, types.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				l.Error("Unable to delete", "image", i, "error", err)
			}
		}
		il.Clear()

		// Remove any images which have been built
		filter := filters.NewArgs()
		filter.Add("reference", "shipyard.run/localcache/*")

		// check if the image already exists, if so do not rebuild unless force
		sum, err := dt.ImageList(context.Background(), types.ImageListOptions{Filters: filter})
		if err != nil {
			l.Error("Unable to check image cache", "error", err)
			bHasError = true
		}

		for _, i := range sum {
			l.Info("Removing image", "image", i.ID)

			_, err := dt.ImageRemove(context.Background(), i.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				l.Error("Unable to delete", "image", i.ID, "error", err)
				bHasError = true
			}
		}

		l.Info("Removing Docker image cache")
		err = dt.VolumeRemove(context.Background(), utils.FQDNVolumeName("images"), true)
		if err != nil {
			l.Error("Unable to remove cached image volume", "error", err)
			bHasError = true
		}

		hcp := utils.GetBlueprintLocalFolder("")
		l.Info("Removing cached blueprints", "path", hcp)
		err = os.RemoveAll(hcp)
		if err != nil {
			l.Error("Unable to remove cached blueprints", "error", err)
			bHasError = true
		}

		bcp := utils.GetHelmLocalFolder("")
		l.Info("Removing cached Helm charts", "path", bcp)
		err = os.RemoveAll(bcp)
		if err != nil {
			l.Error("Unable to remove cached Helm charts", "error", err)
			bHasError = true
		}

		// delete the releases
		rcp := utils.GetReleasesFolder()
		l.Info("Removing cached releases", "path", rcp)
		err = os.RemoveAll(rcp)
		if err != nil {
			l.Error("Unable to remove cached Releases", "error", err)
			bHasError = true
		}

		dcp := utils.GetDataFolder("", os.ModePerm)
		l.Info("Removing data folders", "path", dcp)
		err = os.RemoveAll(dcp)
		if err != nil {
			l.Error("Unable to remove data folder", "error", err)
			bHasError = true
		}

		cp := path.Join(utils.ShipyardHome(), "config")
		l.Info("Removing config", "path", cp)
		err = os.RemoveAll(cp)
		if err != nil {
			l.Error("Unable to remove config folder", "error", err)
			bHasError = true
		}

		if bHasError {
			return fmt.Errorf("An error occured when purging data")
		}

		return nil
	}
}
