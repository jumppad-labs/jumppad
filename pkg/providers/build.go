package providers

import (
	"fmt"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
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
	if b.config.Container.Tag == "" {
		b.config.Container.Tag = "latest"
	}

	// calculate the hash
	hash, err := dirhash.HashDir(b.config.Container.Context, "", dirhash.DefaultHash)
	if err != nil {
		return xerrors.Errorf("unable to hash directory: %w", err)
	}

	b.log.Info(
		"Building image",
		"context", b.config.Container.Context,
		"dockerfile", b.config.Container.DockerFile,
		"image", fmt.Sprintf("jumppad.dev/localcache/%s:%s", b.config.Name, b.config.Container.Tag),
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
	b.log.Info("Checking changes", "ref", b.config.ID)
	return b.hasChanged()
}

func (b *Build) hasChanged() (bool, error) {
	hash, err := dirhash.HashDir(b.config.Container.Context, "", dirhash.DefaultHash)
	if err != nil {
		return false, xerrors.Errorf("unable to hash directory: %w", err)
	}

	if hash != b.config.BuildChecksum {
		return true, nil
	}

	return false, nil
}
