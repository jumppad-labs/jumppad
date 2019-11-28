package providers

import (
	"context"

	"github.com/docker/docker/api/types/container"
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

	dc := &container.Config{}
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	_, err := c.client.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		c.config.Name,
	)

	return err
}
