package providers

import (
	"fmt"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

// ExecRemote provider allows the execution of arbitrary commands on an existing target or
// can create a new container before running
type ExecRemote struct {
	config *config.ExecRemote
	client clients.ContainerTasks
	log    hclog.Logger
}

// NewRemoteExec creates a new Exec provider
func NewRemoteExec(c *config.ExecRemote, ex clients.ContainerTasks, l hclog.Logger) *ExecRemote {
	return &ExecRemote{c, ex, l}
}

// Create a new execution instance
func (c *ExecRemote) Create() error {
	c.log.Info("Remote executing command", "ref", c.config.Name, "command", c.config.Command, "args", c.config.Arguments, "image", c.config.Image)

	/*
		if c.config.Script != "" {
			return fmt.Errorf("Remote execution of Scripts are not currently implemented: %s", c.config.Script)
		}
	*/

	// execution target id
	targetID := ""

	if c.config.Target == "" {
		// Not using existing target create new container
		id, err := c.createRemoteExecContainer()
		if err != nil {
			return xerrors.Errorf("Unable to create container for remote exec: %w", err)
		}

		targetID = id
	} else {
		// Fetch the id for the target
		target, err := c.config.FindDependentResource(c.config.Target)
		if err != nil {
			// this should never happen
			return xerrors.Errorf("Unable to find target: %w", err)
		}

		switch target.Info().Type {
		case config.TypeK8sCluster:
			fallthrough
		case config.TypeNomadCluster:
			fallthrough
		case config.TypeContainer:
			ids, err := c.client.FindContainerIDs(target.Info().Name, target.Info().Type)

			if err != nil {
				return xerrors.Errorf("Unable to find remote exec target: %w", err)
			}

			if len(ids) != 1 {
				return xerrors.Errorf("Unable to find remote exec target")
			}

			targetID = ids[0]
		}
	}

	// execute the script in the container
	command := []string{}
	command = append(command, c.config.Command)
	command = append(command, c.config.Arguments...)

	// build the environment variables
	envs := []string{}
	for _, e := range c.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", e.Key, e.Value))
	}

	for k, v := range c.config.EnvVar {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	err := c.client.ExecuteCommand(targetID, command, envs, c.config.WorkingDirectory, c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
	if err != nil {
		err = xerrors.Errorf("Unable to execute command in remote container: %w", err)
	}

	// destroy the container if we created one
	if c.config.Target == "" {
		c.client.RemoveContainer(targetID)
	}

	return err
}

func (c *ExecRemote) createRemoteExecContainer() (string, error) {
	// generate the ID for the new container based on the clock time and a string
	cc := config.NewContainer(fmt.Sprintf("%d.remote_exec", time.Now().Nanosecond()))
	c.config.ResourceInfo.AddChild(cc)

	cc.Networks = c.config.Networks
	cc.Image = *c.config.Image
	cc.Command = []string{"tail", "-f", "/dev/null"} // ensure container does not immediately exit
	cc.Volumes = c.config.Volumes

	// pull any images needed for this container
	err := c.client.PullImage(cc.Image, false)
	if err != nil {
		c.log.Error("Error pulling container image", "ref", cc.Name, "image", cc.Image.Name)

		return "", err
	}

	return c.client.CreateContainer(cc)
}

// Destroy statisfies the interface requirements but is not used
func (c *ExecRemote) Destroy() error {
	return nil
}

// Lookup statisfies the interface requirements but is not used
func (c *ExecRemote) Lookup() ([]string, error) {
	return []string{}, nil
}
