package container

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/http"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// Container is a provider for creating and destroying Docker containers
type Provider struct {
	config     *Container
	sidecar    *Sidecar
	client     container.ContainerTasks
	httpClient http.HTTP
	log        logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l logger.Logger) error {
	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.client = cli.ContainerTasks
	p.httpClient = cli.HTTP
	p.log = l

	cs, sok := cfg.(*Sidecar)
	if sok {
		co := &Container{}
		co.ResourceMetadata = cs.ResourceMetadata
		co.FQRN = cs.FQRN

		co.Networks = []NetworkAttachment{{ID: cs.Target.FQRN}}
		co.Volumes = cs.Volumes
		co.Command = cs.Command
		co.Entrypoint = cs.Entrypoint
		co.Environment = cs.Environment
		co.HealthCheck = cs.HealthCheck
		co.Image = &cs.Image
		co.Privileged = cs.Privileged
		co.Resources = cs.Resources
		co.MaxRestartCount = cs.MaxRestartCount

		p.sidecar = cs
		p.config = co
		return nil
	}

	c, cok := cfg.(*Container)
	if cok {
		p.config = c
		return nil
	}

	return fmt.Errorf("unable to initialize Container provider, resource is not of type Container or Sidecar")
}

// Create implements provider method and creates a Docker container with the given config
func (p *Provider) Create() error {
	p.log.Info("Creating Container", "ref", p.config.ID)

	err := p.internalCreate(p.sidecar != nil)
	if err != nil {
		return err
	}

	// we need to set the fqdn on the original object
	if p.sidecar != nil {
		p.sidecar.FQRN = p.config.FQRN
	}

	return nil
}

// Lookup the ID based on the config
func (p *Provider) Lookup() ([]string, error) {
	return p.client.FindContainerIDs(p.config.FQRN)
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
	id, err := c.client.FindImageInLocalRegistry(types.Image{Name: c.config.Image.Name})
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

func (c *Provider) internalCreate(sidecar bool) error {
	// set the fqdn
	fqdn := utils.FQDN(c.config.Name, c.config.Module, c.config.Type)
	c.config.FQRN = fqdn

	if c.config.Image == nil {
		return fmt.Errorf("need to specify an image")
	}

	// pull any images needed for this container
	img := types.Image{
		Name:     c.config.Image.Name,
		Username: c.config.Image.Username,
		Password: c.config.Image.Password,
	}

	err := c.client.PullImage(img, false)
	if err != nil {
		c.log.Error("Error pulling container image", "ref", c.config.ID, "image", c.config.Image.Name)

		return err
	}

	// update the image ID
	id, err := c.client.FindImageInLocalRegistry(img)
	if err != nil {
		c.log.Error("Unable to lookup image in local registry", "ref", c.config.ID, "error", err)
		return err
	}

	// id should never be blank here as we have pulled the image
	c.config.Image.ID = id

	new := types.Container{
		Name:            fqdn,
		Image:           &img,
		Entrypoint:      c.config.Entrypoint,
		Command:         c.config.Command,
		Environment:     c.config.Environment,
		DNS:             c.config.DNS,
		Privileged:      c.config.Privileged,
		MaxRestartCount: c.config.MaxRestartCount,
	}

	for _, v := range c.config.Networks {
		new.Networks = append(new.Networks, types.NetworkAttachment{
			ID:          v.ID,
			Name:        v.Name,
			IPAddress:   v.IPAddress,
			Aliases:     v.Aliases,
			IsContainer: sidecar,
		})
	}

	for _, v := range c.config.Volumes {
		new.Volumes = append(new.Volumes, types.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
		})
	}

	for _, p := range c.config.Ports {
		new.Ports = append(new.Ports, types.Port{
			Local:         p.Local,
			Remote:        p.Remote,
			Host:          p.Host,
			Protocol:      p.Protocol,
			OpenInBrowser: p.OpenInBrowser,
		})
	}

	for _, pr := range c.config.PortRanges {
		new.PortRanges = append(new.PortRanges, types.PortRange{
			Range:      pr.Range,
			EnableHost: pr.EnableHost,
			Protocol:   pr.Protocol,
		})
	}

	if c.config.Resources != nil {
		new.Resources = &types.Resources{
			CPU:    c.config.Resources.CPU,
			CPUPin: c.config.Resources.CPUPin,
			Memory: c.config.Resources.Memory,
		}
	}

	if c.config.RunAs != nil {
		new.RunAs = &types.User{
			User:  c.config.RunAs.User,
			Group: c.config.RunAs.Group,
		}
	}

	id, err = c.client.CreateContainer(&new)
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
				c.config.Networks[i].AssignedAddress = n.IPAddress
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
