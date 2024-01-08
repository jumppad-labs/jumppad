package exec

import (
	"fmt"
	"path/filepath"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/command"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/command/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// ExecLocal provider allows the execution of arbitrary commands
// on the local machine
type LocalProvider struct {
	config *LocalExec
	client command.Command
	log    sdk.Logger
}

// Intit creates a new Local Exec provider
func (p *LocalProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*LocalExec)
	if !ok {
		return fmt.Errorf("unable to initialize Local provider, resource is not of type LocalExec")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.Command
	p.log = l

	return nil
}

// Create a new exec
func (p *LocalProvider) Create() error {
	p.log.Info("Locally executing script", "ref", p.config.ResourceID, "command", p.config.Command)
	p.log.Warn("This resource is deprecated and will be removed in a future version of Jumppad, please use exec instead")

	// build the environment variables
	envs := []string{}

	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	// create the folders for logs and pids
	logPath := filepath.Join(utils.LogsDir(), fmt.Sprintf("exec_%s.log", p.config.ResourceName))

	// do we have a duration to parse
	var d time.Duration
	var err error
	if p.config.Timeout != "" {
		d, err = time.ParseDuration(p.config.Timeout)
		if err != nil {
			return fmt.Errorf("unable to parse Duration for timeout: %s", err)
		}

		if p.config.Daemon {
			p.log.Warn("timeout will be ignored when exec is running in daemon mode")
		}
	}

	// create the config
	cc := ctypes.CommandConfig{
		Command:          p.config.Command[0],
		Args:             p.config.Command[1:],
		Env:              envs,
		WorkingDirectory: p.config.WorkingDirectory,
		RunInBackground:  p.config.Daemon,
		LogFilePath:      logPath,
		Timeout:          d,
	}

	pid, err := p.client.Execute(cc)

	// set the output
	p.config.Pid = pid

	p.log.Debug("Started process", "ref", p.config.ResourceID, "pid", p.config.Pid)

	if err != nil {
		return err
	}

	return nil
}

// Destroy satisfies the interface method but is not implemented by LocalExec
func (p *LocalProvider) Destroy() error {
	if p.config.Daemon {
		// attempt to destroy the process
		p.log.Info("Stopping locally executing script", "ref", p.config.ResourceID, "pid", p.config.Pid)

		if p.config.Pid < 1 {
			p.log.Warn("Unable to stop local process, no pid")
			return nil
		}

		err := p.client.Kill(p.config.Pid)
		if err != nil {
			p.log.Warn("Error cleaning up daemonized process", "error", err)
		}
	}

	return nil
}

// Lookup satisfies the interface method but is not implemented by LocalExec
func (p *LocalProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *LocalProvider) Refresh() error {
	p.log.Debug("Refresh Local Exec", "ref", p.config.ResourceName)

	return nil
}

func (p *LocalProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ResourceID)

	return false, nil
}
