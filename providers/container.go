package providers

import (
	"context"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config *config.Container
	client clients.Docker
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co *config.Container, cl clients.Docker) *Container {
	return &Container{co, cl}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {

	// create a unique name based on service network [container].[network].shipyard
	// attach to networks
	// - networkRef
	// - wanRef

	dc := &container.Config{
		Hostname:     c.config.Name,
		Image:        c.config.Image,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create volume mounts
	mounts := []mount.Mount{}
	for _, vc := range c.config.Volumes {
		sourcePath, err := filepath.Abs(vc.Source)
		if err != nil {
			return err
		}

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sourcePath,
			Target: vc.Destination,
		})
	}

	hc := &container.HostConfig{
		Mounts: mounts,
	}

	nc := &network.NetworkingConfig{}

	cont, err := c.client.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		FQDN(c.config.Name, c.config.NetworkRef.Name),
	)
	c.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})

	return err
}

func (c *Container) Destroy() error {
	id, err := c.Lookup()
	if err != nil {
		return err
	}

	return c.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
}

func (c *Container) Lookup() (string, error) {
	name := FQDN(c.config.Name, c.config.NetworkRef.Name)

	args, _ := filters.ParseFlag("name="+name, filters.NewArgs())
	args, _ = filters.ParseFlag("status=running", args)

	opts := types.ContainerListOptions{Filters: args}

	cl, err := c.client.ContainerList(context.Background(), opts)
	if err != nil {
		return "", err
	}

	return cl[0].ID, nil
}
