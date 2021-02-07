package clients

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func setupExecute(t *testing.T) Command {
	return NewCommand(3*time.Second, hclog.New(&hclog.LoggerOptions{Level: hclog.Debug}))
}

func TestExecuteWithBasicParams(t *testing.T) {
	e := setupExecute(t)

	err := e.Execute(CommandConfig{Command: "ls"})
	assert.NoError(t, err)
}

func TestExecuteLongRunningTimesOut(t *testing.T) {
	e := setupExecute(t)

	err := e.Execute(CommandConfig{
		Command: "tail",
		Args:    []string{"-f", "/dev/null"},
	})

	assert.Error(t, err)
	assert.Equal(t, ErrorCommandTimeout, err)
}
