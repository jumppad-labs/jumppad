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
	c.log.Info("Remote executing script", "ref", c.config.Name, "script", c.config.Script, "image", c.config.Image)

	if c.config.Script != "" {
		return fmt.Errorf("Remote execution of Scripts is not currently implemented")
	}

	if c.config.Target != "" {
		return fmt.Errorf("Remote execution in existing target is not yet implemented")
	}

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
		return xerrors.Errorf("Unable to create container for remote exec: %w", err)
	}

	// get the id of the new container
	id, err := cp.Lookup()
	if err != nil {
		return xerrors.Errorf("Unable to find id for remote exec container: %w", err)
	}

	if id == "" {
		return xerrors.Errorf("Unable to find id for remote exec container")
	}

	// execute the script in the container
	err = execCommand(c.client, id, []string{c.config.Command}, c.log)
	if err != nil {
		return xerrors.Errorf("Unable to execute command in remote container: %w", err)
	}

	// destroy the container
	return cp.Destroy()
}

// Destroy statisfies the interface requirements but is not used
func (c *RemoteExec) Destroy() error {
	return nil
}

// Lookup statisfies the interface requirements but is not used
func (c *RemoteExec) Lookup() (string, error) {
	return "", nil
}
