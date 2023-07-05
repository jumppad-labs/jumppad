package providers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig/types"
)

// Null is a noop provider
type Null struct {
	config types.Resource
	log    hclog.Logger
}

// NewNull creates a null noop provider
func NewNull(c types.Resource, l hclog.Logger) *Null {
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
