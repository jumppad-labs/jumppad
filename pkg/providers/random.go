package providers

import (
	"math/rand"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
)

// Null is a noop provider
type RandomNumber struct {
	config *resources.RandomNumber
	log    hclog.Logger
}

// NewNull creates a null noop provider
func NewRandomNumber(c *resources.RandomNumber, l hclog.Logger) *RandomNumber {
	return &RandomNumber{c, l}
}

func (n *RandomNumber) Create() error {
	n.log.Info("Creating random number", "ref", n.config.Metadata().ID)

	rn := rand.Intn(n.config.Maximum-n.config.Minimum) + n.config.Minimum
	n.log.Debug("Generated random number", "ref", n.config.Metadata().ID, "number", rn)

	n.config.Value = rn

	return nil
}

func (n *RandomNumber) Destroy() error {
	return nil
}

func (n *RandomNumber) Lookup() ([]string, error) {
	return nil, nil
}

func (n *RandomNumber) Refresh() error {
	return nil
}
