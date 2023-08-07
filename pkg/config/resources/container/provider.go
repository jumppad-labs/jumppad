package container

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// Container is a provider for creating and destroying Docker containers
type Provider struct {
	config     *Container
	sidecar    *Sidecar
	client     container.ContainerTasks
	httpClient clients.HTTP
	log        clients.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewContainerProvider(co *Container, cl clients.ContainerTasks, hc clients.HTTP, l clients.Logger) *Provider {
	return &Provider{config: co, client: cl, httpClient: hc, log: l}
}

func NewSidecarProvider(cs *Sidecar, cl clients.ContainerTasks, hc clients.HTTP, l clients.Logger) *Provider {
	co := &Container{}
	co.ResourceMetadata = cs.ResourceMetadata
	co.FQRN = cs.FQRN

	co.Networks = []NetworkAttachment{NetworkAttachment{ID: cs.Target}}
	co.Volumes = cs.Volumes
	co.Command = cs.Command
	co.Entrypoint = cs.Entrypoint
	co.Environment = cs.Environment
	co.HealthCheck = cs.HealthCheck
	co.Image = &cs.Image
	co.Privileged = cs.Privileged
	co.Resources = cs.Resources
	co.MaxRestartCount = cs.MaxRestartCount

	return &Provider{config: co, client: cl, httpClient: hc, log: l, sidecar: cs}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Provider) Create() error {
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
func (c *Provider) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(c.config.FQRN)
}

func (c *Provider) Refresh() error {
	changed, err := c.Changed()
	if err != nil {
		return err
	}

	if changed {
		c.log.Debug("Refresh Container", "ref", c.config.ID)
		err := c.Destroy()
		if err != nil {
			return err
		}

		return c.Create()
	}

	return nil
}

// Destroy stops and removes the container
func (c *Provider) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.ID)

	return c.internalDestroy()
}

func (c *Provider) Changed() (bool, error) {
	// has the image id changed
	id, err := c.client.FindImageInLocalRegistry(*c.config.Image)
	if err != nil {
		c.log.Error("Unable to lookup image in local registry", "ref", c.config.ID, "error", err)
		return false, err
	}

	// check that the current registry id for the image is the same
	// as the image that was used to create this container
	if id != c.config.Image.ID {
		c.log.Debug("Container image changed, needs refresh", "ref", c.config.ID)
		return true, nil
	}

	return false, nil
}

func (c *Provider) internalCreate() error {
	// set the fqdn
	fqdn := utils.FQDN(c.config.Name, c.config.Module, c.config.Type)
	c.config.FQRN = fqdn

	if c.config.Image == nil {
		return fmt.Errorf("need to specify an image")
	}

	// pull any images needed for this container
	err := c.client.PullImage(*c.config.Image, false)
	if err != nil {
		c.log.Error("Error pulling container image", "ref", c.config.ID, "image", c.config.Image.Name)

		return err
	}

	// update the image ID
	id, err := c.client.FindImageInLocalRegistry(*c.config.Image)
	if err != nil {
		c.log.Error("Unable to lookup image in local registry", "ref", c.config.ID, "error", err)
		return err
	}

	// id should never be blank here as we have pulled the image
	c.config.Image.ID = id

	id, err = c.client.CreateContainer(c.config)
	if err != nil {
		c.log.Error("Unable to create container", "ref", c.config.ID, "error", err)
		return err
	}

	// get the assigned ip addresses for the container
	dc := c.client.ListNetworks(id)
	for _, n := range dc {
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

	if c.config.HealthCheck.Timeout == "" {
		c.config.HealthCheck.Timeout = "30s"
	}

	timeout, err := time.ParseDuration(c.config.HealthCheck.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse duration for the health check timeout, please specify as a go duration i.e 30s, 1m: %s", err)
	}

	// execute tcp health checks
	for _, hc := range c.config.HealthCheck.TCP {
		err := c.httpClient.HealthCheckTCP(
			hc.Address,
			timeout,
		)

		if err != nil {
			return err
		}
	}

	// execute http health checks
	for _, hc := range c.config.HealthCheck.HTTP {
		err := c.httpClient.HealthCheckHTTP(
			hc.Address,
			hc.Method,
			hc.Headers,
			hc.Body,
			hc.SuccessCodes,
			timeout,
		)

		if err != nil {
			return err
		}
	}

	for _, hc := range c.config.HealthCheck.Exec {
		err := c.runExecHealthCheck(id, hc.Command, hc.Script, hc.ExitCode, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Provider) runExecHealthCheck(id string, command []string, script string, exitCode int, timeout time.Duration) error {
	if len(script) > 0 {
		// write the script to a temp file
		dir, err := os.MkdirTemp(os.TempDir(), "script*")
		if err != nil {
			return fmt.Errorf("unable to create temporary directory for script: %s", err)
		}

		defer os.RemoveAll(dir)
		fn := path.Join(dir, "script.sh")

		err = os.WriteFile(fn, []byte(script), os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to write script to temporary file %s: %s", dir, err)
		}

		// copy the script to the container
		c.client.CopyFileToContainer(id, fn, "/tmp")

		c.log.Debug("Written script to file", "script", script, "file", fn)

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
		res, err := c.client.ExecuteCommand(id, command, []string{}, "/tmp", "", "", int(timeout.Seconds()), &output)
		if err == nil && exitCode == res {
			c.log.Debug("Exec health check success", "command", command, "output", output.String())
			return nil
		}

		c.log.Debug("Exec health check failed, retrying in 10s", "command", command, "output", output.String())

		// back off
		time.Sleep(10 * time.Second)
	}
}

func (c *Provider) internalDestroy() error {
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
