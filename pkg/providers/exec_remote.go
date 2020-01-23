package providers

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

// RemoteExec provider allows the execution of arbitrary commands on an existing target or
// can create a new container before running
type RemoteExec struct {
	config *config.RemoteExec
	client clients.Docker
	log    hclog.Logger
}

// NewRemoteExec creates a new Exec provider
func NewRemoteExec(c *config.RemoteExec, ex clients.Docker, l hclog.Logger) *RemoteExec {
	return &RemoteExec{c, ex, l}
}

// Create a new execution instance
func (c *RemoteExec) Create() error {
	c.log.Info("Remote executing command", "ref", c.config.Name, "command", c.config.Command, "args", c.config.Arguments, "image", c.config.Image)

	if c.config.Script != "" {
		return fmt.Errorf("Remote execution of Scripts are not currently implemented: %s", c.config.Script)
	}

	// execution target id
	targetID := ""
	var targetContainer *Container // created if necessary

	if c.config.TargetRef == nil {
		// Not using existing target create new container
		id, cp, err := c.createRemoteExecContainer()
		if err != nil {
			return err
		}

		targetID = id
		targetContainer = cp
	} else {
		// Fetch the id for the target
		switch v := c.config.TargetRef.(type) {
		case *config.Container:
			cc := &config.Container{
				Name:       v.Name,
				Network:    v.Network,
				NetworkRef: v.NetworkRef,
			}
			cp := NewContainer(cc, c.client, c.log)
			id, err := cp.Lookup()

			if err != nil {
				return xerrors.Errorf("Unable to find remote exec target: %w", err)
			}

			if id == "" {
				return xerrors.Errorf("Unable to find remote exec target")
			}

			targetID = id
		}
	}

	// execute the script in the container
	command := []string{}
	command = append(command, c.config.Command)
	command = append(command, c.config.Arguments...)
	err := execCommand(c.client, targetID, command, c.log)
	if err != nil {
		return xerrors.Errorf("Unable to execute command in remote container: %w", err)
	}

	// destroy the container if we created one
	if c.config.Target == "" {
		return targetContainer.Destroy()
	}

	return nil
}

func (c *RemoteExec) createRemoteExecContainer() (string, *Container, error) {
	// first create a new container
	cc := &config.Container{
		Name:        "remote_exec_temp",
		Image:       *c.config.Image,
		Command:     []string{"tail", "-f", "/dev/null"}, // ensure container does not immediately exit
		Volumes:     c.config.Volumes,
		NetworkRef:  c.config.WANRef, // seems like wan connections are not implemented //TODO fix
		Environment: c.config.Environment,
	}

	cp := NewContainer(cc, c.client, c.log)

	err := cp.Create()
	if err != nil {
		return "", nil, xerrors.Errorf("Unable to create container for remote exec: %w", err)
	}

	// get the id of the new container
	id, err := cp.Lookup()
	if err != nil {
		return "", nil, xerrors.Errorf("Unable to find id for remote exec container: %w", err)
	}

	if id == "" {
		return "", nil, xerrors.Errorf("Unable to find id for remote exec container")
	}

	return id, cp, nil
}

// Destroy statisfies the interface requirements but is not used
func (c *RemoteExec) Destroy() error {
	return nil
}

// Lookup statisfies the interface requirements but is not used
func (c *RemoteExec) Lookup() (string, error) {
	return "", nil
}
