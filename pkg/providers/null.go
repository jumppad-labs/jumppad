package providers

import (
	"fmt"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
)

// Null is a noop provider
type Null struct {
	config types.Resource
	log    clients.Logger
}

// NewNull creates a null noop provider
func NewNull(c types.Resource, l clients.Logger) *Null {
	return &Null{c, l}
}

func (n *Null) Create() error {
	n.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(n.config.Metadata().Type))), "ref", n.config.Metadata().Name)
	return nil
}

func (n *Null) Destroy() error {
	return nil
}

func (n *Null) Lookup() ([]string, error) {
	return nil, nil
}

func (n *Null) Refresh() error {
	return nil
}

func (c *Null) Changed() (bool, error) {
	return false, nil
}
