package providers

import (
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
	return c.client.Execute(c.config.Command, c.config.Arguments...)
}

func (c *Exec) Destroy() error {
	return nil
}

func (c *Exec) Lookup() (string, error) {
	return "", nil
}
