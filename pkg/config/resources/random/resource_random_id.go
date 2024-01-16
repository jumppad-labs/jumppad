package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomID is the resource for generating random IDs
const TypeRandomID string = "random_id"

// allows the generation of random IDs
type RandomID struct {
	types.ResourceMetadata `hcl:",remain"`

	ByteLength int64 `hcl:"byte_length" json:"byte_length"`

	// Output parameters
	Base64 string `hcl:"base64,optional" json:"base64"`
	Hex    string `hcl:"hex,optional" json:"hex"`
	Dec    string `hcl:"dec,optional" json:"dec"`
}

func (c *RandomID) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ResourceID)
		if r != nil {
			state := r.(*RandomID)
			c.Base64 = state.Base64
			c.Hex = state.Hex
			c.Dec = state.Dec
		}
	}

	return nil
}
