package providers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Null is a noop provider
type Null struct {
	config *config.ResourceInfo
	log    hclog.Logger
}

// NewNull creates a null noop provider
func NewNull(c *config.ResourceInfo, l hclog.Logger) *Null {
	return &Null{c, l}
}

func (n *Null) Create() error {
	n.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(n.config.Type))), "ref", n.config.Name)
	return nil
}

func (n *Null) Destroy() error {
	return nil
}

func (n *Null) Lookup() ([]string, error) {
	return nil, nil
}
