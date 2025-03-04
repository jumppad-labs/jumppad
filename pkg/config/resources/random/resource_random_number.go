package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomNumber is the resource for generating random numbers
const TypeRandomNumber string = "random_number"

/*
allows the generation of random numbers

@resource
*/
type RandomNumber struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	Minimum int `hcl:"minimum" json:"minimum"`
	Maximum int `hcl:"maximum" json:"maximum"`

	// Output parameters
	Value int `hcl:"value,optional" json:"value"`
}

func (c *RandomNumber) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*RandomNumber)
			c.Value = state.Value
		}
	}

	return nil
}
