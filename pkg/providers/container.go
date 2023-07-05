package providers

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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
	co.FQRN = cs.FQRN

	co.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{ID: cs.Target}}
	co.Volumes = cs.Volumes
	co.Command = cs.Command
	co.Entrypoint = cs.Entrypoint
	co.Environment = cs.Environment
	co.HealthCheck = cs.HealthCheck
	co.Image = &cs.Image
	co.Privileged = cs.Privileged
	co.Resources = cs.Resources
	co.MaxRestartCount = cs.MaxRestartCount

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
		c.sidecar.FQRN = c.config.FQRN
	}

	return nil
}

// Lookup the ID based on the config
func (c *Container) Lookup() ([]string, error) {
	c.log.Debug("Lookup Container Details", "fqrn", c.config.FQRN)

	return c.client.FindContainerIDs(c.config.FQRN)
}

func (c *Container) Refresh() error {
	c.log.Info("Refresh Container", "ref", c.config.Name)

	return nil
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.ID)

	return c.internalDestroy()
}

func (c *Container) internalCreate() error {

	// set the fqdn
	fqdn := utils.FQDN(c.config.Name, c.config.Module, c.config.Type)
	c.config.FQRN = fqdn

	// do we need to build an image
	if c.config.Build != nil {

		if c.config.Build.Tag == "" {
			c.config.Build.Tag = "latest"
		}

		c.log.Debug(
			"Building image",
			"context", c.config.Build.Context,
			"dockerfile", c.config.Build.DockerFile,
			"image", fmt.Sprintf("jumppad.dev/localcache/%s:%s", c.config.Name, c.config.Build.Tag),
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

	timeout, err := time.ParseDuration(c.config.HealthCheck.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse duration for the health check timeout, please specify as a go duration i.e 30s, 1m: %s", err)
	}

	// check the health of the container
	if c.config.HealthCheck.HTTP != nil {
		err := c.httpClient.HealthCheckHTTP(
			c.config.HealthCheck.HTTP.Address,
			c.config.HealthCheck.HTTP.SuccessCodes,
			timeout,
		)

		if err != nil {
			return err
		}
	}

	if c.config.HealthCheck.TCP != nil {
		err := c.httpClient.HealthCheckTCP(
			c.config.HealthCheck.TCP.Address,
			timeout,
		)

		if err != nil {
			return err
		}
	}

	if c.config.HealthCheck.Exec != nil {
		err := c.runExecHealthCheck(id, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) runExecHealthCheck(id string, timeout time.Duration) error {
	command := []string{}

	if len(c.config.HealthCheck.Exec.Command) > 0 {
		command = c.config.HealthCheck.Exec.Command
	}

	if len(c.config.HealthCheck.Exec.Script) > 0 {
		// write the script to a temp file
		dir, err := os.MkdirTemp(os.TempDir(), "script*")
		if err != nil {
			return fmt.Errorf("unable to create temporary directory for script: %s", err)
		}

		defer os.RemoveAll(dir)
		fn := path.Join(dir, "script.sh")

		err = os.WriteFile(fn, []byte(c.config.HealthCheck.Exec.Script), os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to write script to temporary file %s: %s", dir, err)
		}

		// copy the script to the container
		c.client.CopyFileToContainer(id, fn, "/tmp")

		c.log.Debug("Written script to file", "script", c.config.HealthCheck.Exec.Script, "file", fn)

		command = []string{"sh", "/tmp/script.sh"}
	}

	c.log.Debug("Performing Exec health check with", "command", command)
	st := time.Now()

	for {
		if time.Since(st) > timeout {
			c.log.Error("Timeout waiting for Exec health check")

			return fmt.Errorf("timeout waiting for Exec health check %v", command)
		}

		var output bytes.Buffer
		_, err := c.client.ExecuteCommand(id, command, []string{}, "/tmp", "", "", 300, &output)
		if err == nil {
			c.log.Debug("Exec health check success", "command", command, "output", output.String())
			return nil
		}

		c.log.Debug("Exec health check failed, retrying in 10s", "command", command, "output", output.String())

		// back off
		time.Sleep(10 * time.Second)
	}

	return nil
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
