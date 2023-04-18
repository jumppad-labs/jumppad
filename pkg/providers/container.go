package providers

import (
	"fmt"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config     *resources.Container
	sidecar    *resources.Sidecar
	client     clients.ContainerTasks
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co *resources.Container, cl clients.ContainerTasks, hc clients.HTTP, l hclog.Logger) *Container {
	return &Container{config: co, client: cl, httpClient: hc, log: l}
}

func NewContainerSidecar(cs *resources.Sidecar, cl clients.ContainerTasks, hc clients.HTTP, l hclog.Logger) *Container {
	co := &resources.Container{}
	co.ResourceMetadata = cs.ResourceMetadata

	co.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{ID: cs.Target}}
	co.Volumes = cs.Volumes
	co.Command = cs.Command
	co.Entrypoint = cs.Entrypoint
	co.Env = cs.Env
	co.HealthCheck = cs.HealthCheck
	co.Image = &cs.Image
	co.Privileged = cs.Privileged
	co.Resources = cs.Resources
	co.MaxRestartCount = cs.MaxRestartCount
	co.FQDN = cs.FQDN

	return &Container{config: co, client: cl, httpClient: hc, log: l, sidecar: cs}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {
	c.log.Info("Creating Container", "ref", c.config.ID)

	err := c.internalCreate()
	if err != nil {
		return err
	}

	// we need to set the fqdn on the original object
	if c.sidecar != nil {
		c.sidecar.FQDN = c.config.FQDN
	}

	return nil
}

func (c *Container) internalCreate() error {
	// do we need to build an image
	if c.config.Build != nil {

		if c.config.Build.Tag == "" {
			c.config.Build.Tag = "latest"
		}

		c.log.Debug(
			"Building image",
			"context", c.config.Build.Context,
			"dockerfile", c.config.Build.DockerFile,
			"image", fmt.Sprintf("shipyard.run/localcache/%s:%s", c.config.Name, c.config.Build.Tag),
		)

		name, err := c.client.BuildContainer(c.config, false)
		if err != nil {
			return xerrors.Errorf("Unable to build image: %w", err)
		}

		// set the image to be loaded and continue with the container creation
		c.config.Image = &resources.Image{Name: name}
	} else {
		// pull any images needed for this container
		err := c.client.PullImage(*c.config.Image, false)
		if err != nil {
			c.log.Error("Error pulling container image", "ref", c.config.ID, "image", c.config.Image.Name)

			return err
		}
	}

	id, err := c.client.CreateContainer(c.config)
	if err != nil {
		c.log.Error("Unable to create container", "ref", c.config.ID, "error", err)
		return err
	}

	// set the fqdn
	fqdn := utils.FQDN(c.config.Name, c.config.Module, c.config.Type)
	c.config.FQDN = fqdn

	// get the assigned ip addresses for the container
	dc := c.client.ListNetworks(id)
	for _, n := range dc {
		c.log.Info("network", "net", n)
		for i, net := range c.config.Networks {
			if net.ID == n.ID {
				// set the assigned address and name
				c.config.Networks[i].AssignedAddress = n.AssignedAddress
				c.config.Networks[i].Name = n.Name
			}
		}
	}

	if c.config.HealthCheck == nil {
		return nil
	}

	// check the health of the container
	if hc := c.config.HealthCheck.HTTP; hc != "" {
		d, err := time.ParseDuration(c.config.HealthCheck.Timeout)
		if err != nil {
			return err
		}

		// do we have custom status codes, if not use 200
		codes := c.config.HealthCheck.HTTPSuccessCodes
		if codes == nil {
			codes = []int{200}
		}

		return c.httpClient.HealthCheckHTTP(hc, codes, d)
	}

	return nil
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.ID)

	return c.internalDestroy()
}

func (c *Container) internalDestroy() error {
	ids, err := c.Lookup()
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		for _, id := range ids {
			err := c.client.RemoveContainer(id, false)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Lookup the ID based on the config
func (c *Container) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(c.config.FQDN)
}
