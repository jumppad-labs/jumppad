package providers

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Exec provider allows the execution of arbitrary commands
type Exec struct {
	config *config.Exec
	client clients.Command
	log    hclog.Logger
}

// NewExec creates a new Exec provider
func NewExec(c *config.Exec, ex clients.Command, l hclog.Logger) *Exec {
	return &Exec{c, ex, l}
}

func (c *Exec) Create() error {
	c.log.Debug("Executing script", "ref", c.config.Name, "script", c.config.Script)

	// make sure the script is executable
	err := os.Chmod(c.config.Script, 0777)
	if err != nil {
		c.log.Error("Unable to set script permissions", "error", err)
	}

	return c.client.Execute(c.config.Script)
}

func (c *Exec) Destroy() error {
	return nil
}

func (c *Exec) Lookup() (string, error) {
	return "", nil
}
