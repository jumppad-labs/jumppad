package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomPassword is the resource for generating random passwords
const TypeRandomPassword string = "random_password"

// allows the generation of random Passwords
type RandomPassword struct {
	types.ResourceBase `hcl:",remain"`

	Length int64 `hcl:"length" json:"lenght"`

	OverrideSpecial string `hcl:"override_special,optional" json:"override_special"`

	Special    *bool `hcl:"special,optional" json:"special"`
	Numeric    *bool `hcl:"numeric,optional" json:"numeric"`
	Lower      *bool `hcl:"lower,optional" json:"lower"`
	Upper      *bool `hcl:"upper,optional" json:"upper"`
	MinSpecial int64 `hcl:"min_special,optional" json:"min_special"`
	MinNumeric int64 `hcl:"min_numeric,optional" json:"min_numeric"`
	MinLower   int64 `hcl:"min_lower,optional" json:"min_lower"`
	MinUpper   int64 `hcl:"min_upper,optional" json:"min_upper"`

	// Output parameters
	Value string `hcl:"value,optional" json:"value"`
}

func (c *RandomPassword) Process() error {
	if c.Special == nil {
		c.Special = boolPointer(true)
	}

	if c.Numeric == nil {
		c.Numeric = boolPointer(true)
	}

	if c.Lower == nil {
		c.Lower = boolPointer(true)
	}

	if c.Upper == nil {
		c.Upper = boolPointer(true)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*RandomPassword)
			c.Value = state.Value
		}
	}

	return nil
}
