package cmd

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/docker/docker/api/types/filters"
	dimage "github.com/docker/docker/api/types/image"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/images"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newPurgeCmd(dt container.Docker, il images.ImageLog, l logger.Logger) *cobra.Command {
	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Purges Docker images, Helm charts, and Blueprints downloaded by jumppad",
		Long:  "Purges Docker images, Helm charts, and Blueprints downloaded by jumppad",
		Example: `
  jumppad purge
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newPurgeCmdFunc(dt, il, l),
		SilenceUsage: true,
	}

	return purgeCmd
}

func newPurgeCmdFunc(dt container.Docker, il images.ImageLog, l logger.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		images, _ := il.Read(images.ImageTypeDocker)

		bHasError := false

		for _, i := range images {
			l.Info("Removing image", "image", i)

			_, err := dt.ImageRemove(context.Background(), i, dimage.RemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				l.Error("Unable to delete", "image", i, "error", err)
			}
		}
		il.Clear()

		// Remove any images which have been built
		filter := filters.NewArgs()
		filter.Add("reference", "jumppad.dev/localcache/*")

		// check if the image already exists, if so do not rebuild unless force
		sum, err := dt.ImageList(context.Background(), dimage.ListOptions{Filters: filter})
		if err != nil {
			l.Error("Unable to check image cache", "error", err)
			bHasError = true
		}

		for _, i := range sum {
			l.Info("Removing image", "image", i.ID)

			_, err := dt.ImageRemove(context.Background(), i.ID, dimage.RemoveOptions{Force: true, PruneChildren: true})
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

		hcp := utils.BlueprintLocalFolder("")
		l.Info("Removing cached blueprints", "path", hcp)
		err = os.RemoveAll(hcp)
		if err != nil {
			l.Error("Unable to remove cached blueprints", "error", err)
			bHasError = true
		}

		bcp := utils.HelmLocalFolder("")
		l.Info("Removing cached Helm charts", "path", bcp)
		err = os.RemoveAll(bcp)
		if err != nil {
			l.Error("Unable to remove cached Helm charts", "error", err)
			bHasError = true
		}

		// delete the releases
		rcp := utils.ReleasesFolder()
		l.Info("Removing cached releases", "path", rcp)
		err = os.RemoveAll(rcp)
		if err != nil {
			l.Error("Unable to remove cached Releases", "error", err)
			bHasError = true
		}

		dcp := utils.DataFolder("", os.ModePerm)
		l.Info("Removing data folders", "path", dcp)
		err = os.RemoveAll(dcp)
		if err != nil {
			l.Error("Unable to remove data folder", "error", err)
			bHasError = true
		}

		ccp := utils.DataFolder("", os.ModePerm)
		l.Info("Removing cache folders", "path", ccp)
		err = os.RemoveAll(ccp)
		if err != nil {
			l.Error("Unable to remove cache folder", "error", err)
			bHasError = true
		}

		cp := path.Join(utils.JumppadHome(), "config")
		l.Info("Removing config", "path", cp)
		err = os.RemoveAll(cp)
		if err != nil {
			l.Error("Unable to remove config folder", "error", err)
			bHasError = true
		}

		if bHasError {
			return fmt.Errorf("an error occurred when purging data")
		}

		return nil
	}
}
