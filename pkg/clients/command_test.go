package clients

import (
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func setupExecute(t *testing.T) Command {
	return NewCommand(3*time.Second, hclog.NewNullLogger())
}

func TestExecuteWithBasicParams(t *testing.T) {
	command := "sh"
	args := []string{"ls"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "dir"}
	}

	e := setupExecute(t)

	err := e.Execute(CommandConfig{
		Command: command,
		Args:    args,
	})

	assert.NoError(t, err)
}

func TestExecuteLongRunningTimesOut(t *testing.T) {
	command := "sh"
	args := []string{"sleep", "10"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "ping", "192.0.2.1", "-n", "1", "-w", "100000", ">NUL"}
	}

	e := setupExecute(t)

	err := e.Execute(CommandConfig{
		Command: command,
		Args:    args,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrorCommandTimeout, err)
}

func TestExecuteInvalidCommandReturnsError(t *testing.T) {
	e := setupExecute(t)

	err := e.Execute(CommandConfig{Command: "nocommand"})
	assert.Error(t, err)
}
