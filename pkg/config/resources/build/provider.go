package build

import (
	"context"
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// Null is a noop provider
type Provider struct {
	config *Build
	client container.ContainerTasks
	log    sdk.Logger
}

// NewBuild creates a null noop provider
func (b *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Build)
	if !ok {
		return fmt.Errorf("unable to initialize Build provider, resource is not of type Build")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	b.config = c
	b.client = cli.ContainerTasks
	b.log = l

	return nil
}

func (b *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		b.log.Debug("Context cancelled, skipping build", "ref", b.config.Meta.ID)
		return nil
	}

	// calculate the hash
	hash, err := utils.HashDir(b.config.Container.Context, b.config.Container.Ignore...)
	if err != nil {
		return fmt.Errorf("unable to hash directory: %w", err)
	}

	tag, _ := utils.ReplaceNonURIChars(hash[3:11])

	b.log.Info(
		"Building image",
		"context", b.config.Container.Context,
		"dockerfile", b.config.Container.DockerFile,
		"image", fmt.Sprintf("jumppad.dev/localcache/%s:%s", b.config.Meta.Name, tag),
	)

	force := false
	if hash != b.config.BuildChecksum {
		force = true
	}

	build := &types.Build{
		Name:       b.config.Meta.Name,
		DockerFile: b.config.Container.DockerFile,
		Context:    b.config.Container.Context,
		Ignore:     b.config.Container.Ignore,
		Args:       b.config.Container.Args,
	}

	name, err := b.client.BuildContainer(build, force)
	if err != nil {
		return fmt.Errorf("unable to build image: %w", err)
	}

	// set the image to be loaded and continue with the container creation
	b.config.Image = name
	b.config.BuildChecksum = hash

	// do we need to copy any files?
	err = b.copyOutputs()
	if err != nil {
		return fmt.Errorf("unable to copy files from build container: %w", err)
	}

	// clean up the previous builds only leaving the last 3
	ids, err := b.client.FindImagesInLocalRegistry(fmt.Sprintf("jumppad.dev/localcache/%s", b.config.Meta.Name))
	if err != nil {
		return fmt.Errorf("unable to query local registry for images: %w", err)
	}

	for i := 3; i < len(ids); i++ {
		b.log.Debug("Remove image", "ref", b.config.Meta.ID, "id", ids[i])

		err := b.client.RemoveImage(ids[i])
		if err != nil {
			return fmt.Errorf("unable to remove old build images: %w", err)
		}
	}

	// if we have a registry, push the image
	for _, r := range b.config.Registries {
		// first tag the image
		b.log.Debug("Tag image", "ref", b.config.Meta.ID, "name", b.config.Image, "tag", r.Name)
		err = b.client.TagImage(b.config.Image, r.Name)
		if err != nil {
			return fmt.Errorf("unable to tag image: %w", err)
		}

		// push the image
		b.log.Debug("Push image", "ref", b.config.Meta.ID, "tag", r.Name)
		err = b.client.PushImage(types.Image{Name: r.Name, Username: r.Username, Password: r.Password})
		if err != nil {
			return fmt.Errorf("unable to push image: %w", err)
		}
	}

	return nil
}

func (b *Provider) Destroy(ctx context.Context, force bool) error {
	b.log.Info("Destroy Build", "ref", b.config.Meta.ID)

	return nil
}

func (b *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (b *Provider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		b.log.Debug("Context cancelled, skipping refresh", "ref", b.config.Meta.ID)
		return nil
	}

	// calculate the hash
	changed, err := b.hasChanged()
	if err != nil {
		return err
	}

	if changed {
		b.log.Info("Build status changed, rebuild")
		err := b.Destroy(ctx, false)
		if err != nil {
			return fmt.Errorf("unable to destroy existing container: %w", err)
		}

		return b.Create(ctx)
	}
	return nil
}

func (b *Provider) Changed() (bool, error) {
	changed, err := b.hasChanged()
	if err != nil {
		return false, err
	}

	if changed {
		b.log.Debug("Build has changed, requires refresh", "ref", b.config.Meta.ID)
		return true, nil
	}

	return false, nil
}

func (b *Provider) hasChanged() (bool, error) {
	hash, err := utils.HashDir(b.config.Container.Context, b.config.Container.Ignore...)
	if err != nil {
		return false, fmt.Errorf("unable to hash directory: %w", err)
	}

	if hash != b.config.BuildChecksum {
		return true, nil
	}

	return false, nil
}

func (b *Provider) copyOutputs() error {
	if len(b.config.Outputs) < 1 {
		return nil
	}

	// start an instance of the container
	c := types.Container{
		Image: &types.Image{
			Name: b.config.Image,
		},
		Entrypoint: []string{},
		Command:    []string{"tail", "-f", "/dev/null"},
	}

	b.log.Debug("Creating container to copy files", "ref", b.config.Meta.ID, "name", b.config.Image)
	id, err := b.client.CreateContainer(&c)
	if err != nil {
		return err
	}

	// always remove the temp container
	defer func() {
		b.log.Debug("Remove copy container", "ref", b.config.Meta.ID, "name", b.config.Image)
		b.client.RemoveContainer(id, true)
	}()

	for _, copy := range b.config.Outputs {
		b.log.Debug("Copy file from container", "ref", b.config.Meta.ID, "source", copy.Source, "destination", copy.Destination)
		err := b.client.CopyFromContainer(id, copy.Source, copy.Destination)
		if err != nil {
			return err
		}
	}

	return nil
}
