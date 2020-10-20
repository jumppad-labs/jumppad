package providers

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// ExecLocal provider allows the execution of arbitrary commands
// on the local machine
type ExecLocal struct {
	config *config.ExecLocal
	client clients.Command
	log    hclog.Logger
}

// NewExecLocal creates a new Local Exec provider
func NewExecLocal(c *config.ExecLocal, ex clients.Command, l hclog.Logger) *ExecLocal {
	return &ExecLocal{c, ex, l}
}

// Create a new exec
func (c *ExecLocal) Create() error {
	c.log.Info("Locally executing script", "ref", c.config.Name, "script", c.config.Command, "args", c.config.Arguments)

	// build the environment variables
	envs := []string{}
	for _, e := range c.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", e.Key, e.Value))
	}

	for k, v := range c.config.EnvVar {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	// create the config
	cc := clients.CommandConfig{
		Command:          c.config.Command,
		Args:             c.config.Arguments,
		Env:              envs,
		WorkingDirectory: c.config.WorkingDirectory,
	}

	// set the env vars
	err := c.client.Execute(cc)
	if err != nil {
		return err
	}

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Destroy() error {
	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Lookup() ([]string, error) {
	return []string{}, nil
}
