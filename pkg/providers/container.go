package providers

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config config.Container
	client clients.ContainerTasks
	log    hclog.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co config.Container, cl clients.ContainerTasks, l hclog.Logger) *Container {
	return &Container{co, cl, l}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {
	c.log.Info("Creating Container", "ref", c.config.Name)

	// pull any images needed for this container
	err := c.client.PullImage(c.config.Image, false)
	if err != nil {
		c.log.Error("Error pulling container image", "ref", c.config.Name, "image", c.config.Image.Name)

		return err
	}

	_, err = c.client.CreateContainer(c.config)

	// set the state
	c.config.State = config.Applied

	return err
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.Name)
	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.NetworkRef.Name)

	if err != nil {
		return err
	}

	if len(ids) > 0 {
		for _, id := range ids {
			err := c.client.RemoveContainer(id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Config returns the config for the provider
func (c *Container) Config() ConfigWrapper {
	return ConfigWrapper{"config.Container", c.config}
}

// Lookup the ID based on the config
func (c *Container) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(c.config.Name, c.config.NetworkRef.Name)
}

// State returns the state from the config
func (c *Container) State() config.State {
	return c.config.State
}

// SetState updates the state in the config
func (c *Container) SetState(state config.State) {
	c.config.State = state
}
