package providers

import (
	"fmt"

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

	if c.config.Script != "" {
		return fmt.Errorf("Remote execution of Scripts are not currently implemented: %s", c.config.Script)
	}

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

	err := c.client.ExecuteCommand(targetID, command, c.log.StandardWriter(&hclog.StandardLoggerOptions{}))
	if err != nil {
		return xerrors.Errorf("Unable to execute command in remote container: %w", err)
	}

	// destroy the container if we created one
	if c.config.Target == "" {
		return c.client.RemoveContainer(targetID)
	}

	return nil
}

func (c *ExecRemote) createRemoteExecContainer() (string, error) {
	// first create a new container
	cc := config.NewContainer("remote_exec_temp")
	c.config.ResourceInfo.AddChild(cc)

	for _, n := range c.config.Networks {
		cc.Networks = append(cc.Networks, config.NetworkAttachment{Name: n.Name, IPAddress: n.IPAddress})
	}
	cc.Image = *c.config.Image
	cc.Command = []string{"tail", "-f", "/dev/null"} // ensure container does not immediately exit
	cc.Volumes = c.config.Volumes
	cc.Environment = c.config.Environment

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
