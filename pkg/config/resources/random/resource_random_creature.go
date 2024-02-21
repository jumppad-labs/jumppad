package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomCreature is the resource for generating random creatures
const TypeRandomCreature string = "random_creature"

// allows the generation of random creatures
type RandomCreature struct {
	types.ResourceBase `hcl:",remain"`

	// Output parameters
	Value string `hcl:"value,optional" json:"value"`
}

func (c *RandomCreature) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*RandomCreature)
			c.Value = state.Value
		}
	}

	return nil
}

func boolPointer(value bool) *bool {
	return &value
}
