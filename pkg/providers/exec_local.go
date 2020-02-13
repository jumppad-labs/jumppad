package providers

import (
	"fmt"
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// ExecLocal provider allows the execution of arbitrary commands
// on the local machine
type ExecLocal struct {
	config *config.ExecLocal
	client clients.Command
	log    hclog.Logger
}

// NewExecLocal creates a new Local Exec provider
func NewExecLocal(c *config.ExecLocal, ex clients.Command, l hclog.Logger) *ExecLocal {
	return &ExecLocal{c, ex, l}
}

// Create a new exec
func (c *ExecLocal) Create() error {
	if c.config.Command != "" {
		return fmt.Errorf("Only Script execution is currently implemented for Local Exec")
	}

	c.log.Debug("Localy executing script", "ref", c.config.Name, "script", c.config.Script)

	// make sure the script is executable
	err := os.Chmod(c.config.Script, 0777)
	if err != nil {
		c.log.Error("Unable to set script permissions", "error", err)
	}

	err = c.client.Execute(c.config.Script)
	if err != nil {
		return err
	}

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Destroy() error {
	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *ExecLocal) Lookup() ([]string, error) {
	return []string{}, nil
}
