package providers

import (
	"fmt"
	"path/filepath"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
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

	// create the folders for logs and pids
	logPath := filepath.Join(utils.LogsDir(), fmt.Sprintf("exec_%s.log", c.config.Name))

	// do we have a duration to parse
	var d time.Duration
	var err error
	if c.config.Timeout != "" {
		d, err = time.ParseDuration(c.config.Timeout)
		if err != nil {
			return fmt.Errorf("Unable to parse Duration for timeout: %s", err)
		}

		if c.config.Daemon {
			c.log.Warn("Timeout will be ignored when exec is running in daemon mode")
		}
	}

	// create the config
	cc := clients.CommandConfig{
		Command:          c.config.Command,
		Args:             c.config.Arguments,
		Env:              envs,
		WorkingDirectory: c.config.WorkingDirectory,
		RunInBackground:  c.config.Daemon,
		LogFilePath:      logPath,
		Timeout:          d,
	}

	// set the env vars
	p, err := c.client.Execute(cc)
	c.config.Pid = p

	c.log.Debug("Started process", "ref", c.config.Name, "pid", c.config.Pid)

	if err != nil {
		return err
	}

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Destroy() error {
	if c.config.Daemon {
		// attempt to destroy the process
		c.log.Info("Stopping locally executing script", "ref", c.config.Name, "pid", c.config.Pid)

		if c.config.Pid < 1 {
			c.log.Warn("Unable to stop local process, no pid")
			return nil
		}

		return c.client.Kill(c.config.Pid)
	}

	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Lookup() ([]string, error) {
	return []string{}, nil
}
