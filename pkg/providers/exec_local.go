package providers

import (
	"fmt"
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// LocalExec provider allows the execution of arbitrary commands
// on the local machine
type LocalExec struct {
	config *config.LocalExec
	client clients.Command
	log    hclog.Logger
}

// NewLocalExec creates a new LocalExec provider
func NewLocalExec(c *config.LocalExec, ex clients.Command, l hclog.Logger) *LocalExec {
	return &LocalExec{c, ex, l}
}

// Create a new exec
func (c *LocalExec) Create() error {
	if c.config.Command != "" {
		return fmt.Errorf("Only Script execution is currently implemented for Local Exec")
	}

	c.log.Debug("Localy executing script", "ref", c.config.Name, "script", c.config.Script)

	// make sure the script is executable
	err := os.Chmod(c.config.Script, 0777)
	if err != nil {
		c.log.Error("Unable to set script permissions", "error", err)
	}

	return c.client.Execute(c.config.Script)
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *LocalExec) Destroy() error {
	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *LocalExec) Lookup() ([]string, error) {
	return []string{}, nil
}

// Config returns the config for the provider
func (c *LocalExec) Config() ConfigWrapper {
	return ConfigWrapper{"config.LocalExec", c.config}
}
