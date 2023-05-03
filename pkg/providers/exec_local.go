package providers

import (
	"fmt"
	"path/filepath"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// ExecLocal provider allows the execution of arbitrary commands
// on the local machine
type LocalExec struct {
	config *resources.LocalExec
	client clients.Command
	log    hclog.Logger
}

// NewExecLocal creates a new Local Exec provider
func NewLocalExec(c *resources.LocalExec, ex clients.Command, l hclog.Logger) *LocalExec {
	return &LocalExec{c, ex, l}
}

// Create a new exec
func (c *LocalExec) Create() error {
	c.log.Info("Locally executing script", "ref", c.config.Name, "command", c.config.Command)

	// build the environment variables
	envs := []string{}

	for k, v := range c.config.Environment {
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
			return fmt.Errorf("unable to parse Duration for timeout: %s", err)
		}

		if c.config.Daemon {
			c.log.Warn("timeout will be ignored when exec is running in daemon mode")
		}
	}

	// create the config
	cc := clients.CommandConfig{
		Command:          c.config.Command[0],
		Args:             c.config.Command[1:],
		Env:              envs,
		WorkingDirectory: c.config.WorkingDirectory,
		RunInBackground:  c.config.Daemon,
		LogFilePath:      logPath,
		Timeout:          d,
	}

	p, err := c.client.Execute(cc)

	// set the output
	c.config.Pid = p

	c.log.Debug("Started process", "ref", c.config.ID, "pid", c.config.Pid)

	if err != nil {
		return err
	}

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *LocalExec) Destroy() error {
	if c.config.Daemon {
		// attempt to destroy the process
		c.log.Info("Stopping locally executing script", "ref", c.config.Name, "pid", c.config.Pid)

		if c.config.Pid < 1 {
			c.log.Warn("Unable to stop local process, no pid")
			return nil
		}

		err := c.client.Kill(c.config.Pid)
		if err != nil {
			c.log.Warn("Error cleaning up daemonized process", "error", err)
		}
	}

	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *LocalExec) Lookup() ([]string, error) {
	return []string{}, nil
}
