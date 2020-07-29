package providers

import (
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config     *config.Container
	client     clients.ContainerTasks
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co *config.Container, cl clients.ContainerTasks, hc clients.HTTP, l hclog.Logger) *Container {
	return &Container{co, cl, hc, l}
}

func NewContainerSidecar(cs *config.Sidecar, cl clients.ContainerTasks, hc clients.HTTP, l hclog.Logger) *Container {
	co := config.NewContainer(cs.Name)
	co.Depends = cs.Depends
	co.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: cs.Target}}
	co.Volumes = cs.Volumes
	co.Command = cs.Command
	co.Entrypoint = cs.Entrypoint
	co.Environment = cs.Environment
	co.EnvVar = cs.EnvVar
	co.HealthCheck = cs.HealthCheck
	co.Image = &cs.Image
	co.Privileged = cs.Privileged
	co.Resources = cs.Resources
	co.Type = cs.Type
	co.Config = cs.Config

	return &Container{co, cl, hc, l}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {
	c.log.Info("Creating Container", "ref", c.config.Name)

	return c.internalCreate()
}

func (c *Container) internalCreate() error {
	// do we need to build an image
	if c.config.Build != nil {
		c.log.Debug("Building image", "context", c.config.Build.Context, "dockerfile", c.config.Build.File)

		name, err := c.client.BuildContainer(c.config, false)
		if err != nil {
			return xerrors.Errorf("Unable to build image: %w", err)
		}

		// set the image to be loaded and continue with the container creation
		c.config.Image = &config.Image{Name: name}
	} else {
		// pull any images needed for this container
		err := c.client.PullImage(*c.config.Image, false)
		if err != nil {
			c.log.Error("Error pulling container image", "ref", c.config.Name, "image", c.config.Image.Name)

			return err
		}
	}

	_, err := c.client.CreateContainer(c.config)

	if c.config.HealthCheck == nil {
		return err
	}

	// check the health of the container
	if hc := c.config.HealthCheck.HTTP; hc != "" {
		d, err := time.ParseDuration(c.config.HealthCheck.Timeout)
		if err != nil {
			return err
		}

		return c.httpClient.HealthCheckHTTP(hc, d)
	}

	return nil
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.Name)

	return c.internalDestroy()
}

func (c *Container) internalDestroy() error {
	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		for _, id := range ids {
			if c.config.Type == config.TypeContainer {
				for _, n := range c.config.Networks {
					err := c.client.DetachNetwork(n.Name, id)
					if err != nil {
						c.log.Error("Unable to detach network", "ref", c.config.Name, "network", n.Name)
					}
				}
			}

			err := c.client.RemoveContainer(id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Lookup the ID based on the config
func (c *Container) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(c.config.Name, c.config.Type)
}
