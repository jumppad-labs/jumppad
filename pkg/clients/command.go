package clients

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

var ErrorCommandTimeout = fmt.Errorf("Command timed out before completing")

type CommandConfig struct {
	Command          string
	Args             []string
	Env              []string
	WorkingDirectory string
}

type Command interface {
	Execute(config CommandConfig) error
}

// Command executes local commands
type CommandImpl struct {
	timeout time.Duration
	log     hclog.Logger
}

// NewCommand creates a new command with the given logger and maximum command time
func NewCommand(maxCommandTime time.Duration, l hclog.Logger) Command {
	return &CommandImpl{maxCommandTime, l}
}

// Execute the given command
func (c *CommandImpl) Execute(config CommandConfig) error {

	cmd := exec.Command(
		config.Command,
		config.Args...,
	)

	// add the default environment variables
	cmd.Env = os.Environ()

	if config.Env != nil {
		cmd.Env = append(cmd.Env, config.Args...)
	}

	if config.WorkingDirectory != "" {
		cmd.Dir = config.WorkingDirectory
	}

	c.log.Debug("Running command", "cmd", config.Command, "args", config.Args, "dir", config.WorkingDirectory, "env", config.Env)

	// set the standard out and error to the logger
	cmd.Stdout = c.log.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})
	cmd.Stderr = c.log.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})

	// done chan
	done := make(chan error)

	cm := sync.Mutex{}

	// wait for timeout
	t := time.After(c.timeout)

	go func() {
		cm.Lock()

		err := cmd.Start()

		cm.Unlock()

		if err != nil {
			done <- err
		}

		err = cmd.Wait()
		done <- err
	}()

	select {
	case <-t:
		cm.Lock()
		defer cm.Unlock()

		// kill the running process
		cmd.Process.Kill()
		return ErrorCommandTimeout
	case err := <-done:
		return err
	}
}
