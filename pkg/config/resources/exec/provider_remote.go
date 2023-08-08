package exec

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

// ExecRemote provider allows the execution of arbitrary commands on an existing target or
// can create a new container before running
type RemoteProvider struct {
	config *RemoteExec
	client container.ContainerTasks
	log    clients.Logger
}

// NewRemoteExec creates a new Exec provider
func NewRemoteProvider(c *RemoteExec, ex clients.ContainerTasks, l clients.Logger) *RemoteProvider {
	return &RemoteProvider{c, ex, l}
}

// Create a new execution instance
func (c *RemoteExec) Create() error {
	c.log.Info("Remote executing script", "ref", c.config.ID)

	// execution target id
	targetID := ""

	if c.config.Target == "" {
		// Not using existing target create new container
		id, err := c.createRemoteExecContainer()
		if err != nil {
			return xerrors.Errorf("unable to create container for exec_remote.%s: %w", c.config.Name, err)
		}

		targetID = id
	} else {
		// Fetch the id for the target
		target, err := c.config.ParentConfig.FindResource(c.config.Target)
		if err != nil {
			// this should never happen
			return xerrors.Errorf("unable to find target %s: %w", c.config.Target, err)
		}

		switch target.Metadata().Type {
		case resources.TypeK8sCluster:
			fallthrough
		case resources.TypeNomadCluster:
			fallthrough
		case container.TypeSidecar:
			fallthrough
		case container.TypeContainer:
			fqdn := utils.FQDN(target.Metadata().Name, target.Metadata().Module, target.Metadata().Type)
			ids, err := c.client.FindContainerIDs(fqdn)

			if err != nil {
				return xerrors.Errorf("unable to find remote exec target: %w", err)
			}

			if len(ids) != 1 {
				return xerrors.Errorf("unable to find remote exec target %s", fqdn)
			}

			targetID = ids[0]
		}
	}

	// execute the script in the container
	script := c.config.Script

	// build the environment variables
	envs := []string{}

	for k, v := range c.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	user := ""
	group := ""

	if c.config.RunAs != nil {
		user = c.config.RunAs.User
		group = c.config.RunAs.Group
	}

	_, err := c.client.ExecuteScript(targetID, script, envs, c.config.WorkingDirectory, user, group, 300, c.log.StandardWriter())
	if err != nil {
		c.log.Error("Error executing command", "ref", c.config.Name, "image", c.config.Image, "script", c.config.Script)
		err = xerrors.Errorf("Unable to execute command: in remote container: %w", err)
	}

	// destroy the container if we created one
	if c.config.Target == "" {
		c.client.RemoveContainer(targetID, true)
	}

	return err
}

// Destroy satisfies the interface requirements but is not used as the
// resource is not persistent
func (c *RemoteExec) Destroy() error {
	return nil
}

// Lookup satisfies the interface requirements but is not used
// as the resource is not persistent
func (c *RemoteExec) Lookup() ([]string, error) {
	return []string{}, nil
}

func (c *RemoteExec) Refresh() error {
	c.log.Debug("Refresh Remote Exec", "ref", c.config.Name)

	return nil
}

func (c *RemoteExec) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}

func (c *RemoteExec) createRemoteExecContainer() (string, error) {
	// generate the ID for the new container based on the clock time and a string

	cc := &container.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   fmt.Sprintf("%s.remote_exec", c.config.Name),
			Type:   c.config.Type,
			Module: c.config.Module,
		},
	}

	cc.ParentConfig = c.config.Metadata().ParentConfig

	cc.Networks = c.config.Networks
	cc.Image = c.config.Image
	cc.Entrypoint = []string{}
	cc.Command = []string{"tail", "-f", "/dev/null"} // ensure container does not immediately exit
	cc.Volumes = c.config.Volumes

	// pull any images needed for this container
	err := c.client.PullImage(*cc.Image, false)
	if err != nil {
		c.log.Error("Error pulling container image", "ref", cc.Name, "image", cc.Image.Name)

		return "", err
	}

	id, err := c.client.CreateContainer(cc)
	if err != nil {
		c.log.Error("Error creating container for remote exec", "ref", c.config.Name, "image", c.config.Image, "networks", c.config.Networks, "volumes", c.config.Volumes)
		return "", err
	}

	return id, err
}
