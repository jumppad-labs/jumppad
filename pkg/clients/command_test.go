package clients

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

func setupExecute(t *testing.T) Command {
	return NewCommand(30*time.Second, hclog.New(&hclog.LoggerOptions{Level: hclog.Debug}))
}

func TestExecuteWithBasicParams(t *testing.T) {
	t.Skip()
	e := setupExecute(t)

	e.Execute("ls")
}
