package clients

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shipyard-run/gohup"
)

var ErrorCommandTimeout = fmt.Errorf("Command timed out before completing")

type CommandConfig struct {
	Command          string
	Args             []string
	Env              []string
	WorkingDirectory string
	RunInBackground  bool
	LogFilePath      string
	Timeout          time.Duration
}

type Command interface {
	Execute(config CommandConfig) (int, error)
	Kill(pid int) error
}

// Command executes local commands
type CommandImpl struct {
	timeout time.Duration
	log     Logger
}

// NewCommand creates a new command with the given logger and maximum command time
func NewCommand(maxCommandTime time.Duration, l Logger) Command {
	return &CommandImpl{maxCommandTime, l}
}

type done struct {
	pid int
	err error
}

// Execute the given command
func (c *CommandImpl) Execute(config CommandConfig) (int, error) {
	mutex := sync.Mutex{}

	lp := &gohup.LocalProcess{}
	o := gohup.Options{
		Path:    config.Command,
		Args:    config.Args,
		Logfile: config.LogFilePath,
	}

	// add the default environment variables
	o.Env = config.Env

	if config.WorkingDirectory != "" {
		o.Dir = config.WorkingDirectory
	}

	// done chan
	doneCh := make(chan done)

	timeout := c.timeout
	if config.Timeout != (0 * time.Millisecond) {
		timeout = config.Timeout
	}

	// wait for timeout
	t := time.After(timeout)
	var pidfile string
	var pid int
	var err error

	go func() {
		c.log.Debug(
			"Running command",
			"cmd", config.Command,
			"args", config.Args,
			"dir", config.WorkingDirectory,
			"env", config.Env,
			"pid", pidfile,
			"background", config.RunInBackground,
			"log_file", config.LogFilePath,
		)

		mutex.Lock()
		pid, pidfile, err = lp.Start(o)
		if err != nil {
			doneCh <- done{err: err}
		}
		mutex.Unlock()

		// if not background wait for complete
		if !config.RunInBackground {
			for {
				s, err := lp.QueryStatus(pidfile)
				if err != nil {
					doneCh <- done{err: err, pid: pid}
				}

				if s == gohup.StatusStopped {
					break
				}

				time.Sleep(200 * time.Millisecond)
			}
		}

		doneCh <- done{err: err, pid: pid}
	}()

	select {
	case <-t:
		// kill the running process
		mutex.Lock()
		lp.Stop(pidfile)
		mutex.Unlock()
		return pid, ErrorCommandTimeout
	case d := <-doneCh:
		return d.pid, d.err
	}
}

// Kill a process with the given pid
func (c *CommandImpl) Kill(pid int) error {
	lp := gohup.LocalProcess{}
	pidPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d.pid", pid))

	if s, _ := lp.QueryStatus(pidPath); s == gohup.StatusRunning {
		return lp.Stop(pidPath)
	}

	return nil
}
