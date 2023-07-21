package providers

import (
	"fmt"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/xerrors"
)

// Null is a noop provider
type Build struct {
	config *resources.Build
	client clients.ContainerTasks
	log    clients.Logger
}

// NewBuild creates a null noop provider
func NewBuild(cfg *resources.Build, cli clients.ContainerTasks, l clients.Logger) *Build {
	return &Build{cfg, cli, l}
}

func (b *Build) Create() error {
	// calculate the hash
	hash, err := dirhash.HashDir(b.config.Container.Context, "", dirhash.DefaultHash)
	if err != nil {
		return xerrors.Errorf("unable to hash directory: %w", err)
	}

	tag, _ := utils.ReplaceNonURIChars(hash[3:11])

	b.log.Info(
		"Building image",
		"context", b.config.Container.Context,
		"dockerfile", b.config.Container.DockerFile,
		"image", fmt.Sprintf("jumppad.dev/localcache/%s:%s", b.config.Name, tag),
	)

	force := false
	if hash != b.config.Checksum {
		force = true
	}

	name, err := b.client.BuildContainer(b.config, force)
	if err != nil {
		return xerrors.Errorf("unable to build image: %w", err)
	}

	// set the image to be loaded and continue with the container creation
	b.config.Image = name
	b.config.BuildChecksum = hash

	// clean up the previous builds only leaving the last 3
	ids, err := b.client.FindImagesInLocalRegistry(fmt.Sprintf("jumppad.dev/localcache/%s", b.config.Name))
	if err != nil {
		return xerrors.Errorf("unable to query local registry for images: %w", err)
	}

	for i := 3; i < len(ids); i++ {
		b.log.Debug("Remove image", "ref", b.config.ID, "id", ids[i])

		err := b.client.RemoveImage(ids[i])
		if err != nil {
			return xerrors.Errorf("unable to remove old build images: %w", err)
		}
	}

	return nil
}

func (b *Build) Destroy() error {
	b.log.Info("Destroy Build", "ref", b.config.ID)

	return nil
}

func (b *Build) Lookup() ([]string, error) {
	return nil, nil
}

func (b *Build) Refresh() error {
	// calculate the hash
	changed, err := b.hasChanged()
	if err != nil {
		return err
	}

	if changed {
		b.log.Info("Build status changed, rebuild")
		err := b.Destroy()
		if err != nil {
			return xerrors.Errorf("unable to destroy existing container: %w", err)
		}

		return b.Create()
	}
	return nil
}

func (b *Build) Changed() (bool, error) {
	changed, err := b.hasChanged()
	if err != nil {
		return false, err
	}

	if changed {
		b.log.Debug("Build has changed, requires refresh", "ref", b.config.ID)
		return true, nil
	}

	return false, nil
}

func (b *Build) hasChanged() (bool, error) {
	hash, err := utils.HashDir(b.config.Container.Context)
	if err != nil {
		return false, xerrors.Errorf("unable to hash directory: %w", err)
	}

	if hash != b.config.BuildChecksum {
		return true, nil
	}

	return false, nil
}
