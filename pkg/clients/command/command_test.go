package command

import (
	"runtime"
	"testing"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/command/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/stretchr/testify/assert"
)

func setupExecute(t *testing.T) Command {
	return NewCommand(3*time.Second, logger.NewTestLogger(t))
}

func TestExecuteForgroundWithBasicParams(t *testing.T) {
	command := "sh"
	args := []string{"-c", "sleep 1s"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "dir"}
	}

	e := setupExecute(t)

	p, err := e.Execute(types.CommandConfig{
		Command: command,
		Args:    args,
	})

	assert.NoError(t, err)
	assert.Greater(t, p, 1)
}

func TestExecuteForgroundLongRunningTimesOut(t *testing.T) {
	command := "sh"
	args := []string{"-c", "sleep 10s"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "ping", "192.0.2.1", "-n", "1", "-w", "100000", ">NUL"}
	}

	e := setupExecute(t)

	p, err := e.Execute(types.CommandConfig{
		Command: command,
		Args:    args,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrorCommandTimeout, err)
	assert.Greater(t, p, 1)
}

func TestExecuteInvalidCommandReturnsError(t *testing.T) {
	e := setupExecute(t)

	_, err := e.Execute(types.CommandConfig{Command: "nocommand"})
	assert.Error(t, err)
}

func TestExecuteBackgroundWithBasicParams(t *testing.T) {
	command := "sh"
	args := []string{"-c", "sleep 10s"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "dir"}
	}

	e := setupExecute(t)

	timer := time.After(1 * time.Second)
	doneCh := make(chan done)

	go func() {
		p, err := e.Execute(types.CommandConfig{
			Command:         command,
			Args:            args,
			RunInBackground: true,
		})

		doneCh <- done{err: err, pid: p}
	}()

	select {
	case <-timer:
		t.Fatal("Timeout recieved expected command to complete")
	case d := <-doneCh:
		assert.NoError(t, d.err)
		assert.Greater(t, d.pid, 1)
	}
}

func TestKillRemovesProcessWhenRunning(t *testing.T) {
	command := "sh"
	args := []string{"-c", "sleep 10s"}

	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/c", "ping", "192.0.2.1", "-n", "1", "-w", "100000", ">NUL"}
	}

	e := setupExecute(t)

	timer := time.After(1 * time.Second)
	doneCh := make(chan done)

	go func() {
		p, err := e.Execute(types.CommandConfig{
			Command:         command,
			Args:            args,
			RunInBackground: true,
		})

		doneCh <- done{err: err, pid: p}
	}()

	select {
	case <-timer:
		t.Fatal("Timeout recieved expected command to complete")
	case d := <-doneCh:
		assert.NoError(t, d.err)
		assert.Greater(t, d.pid, 1)

		err := e.Kill(d.pid)
		assert.NoError(t, err)
	}
}
