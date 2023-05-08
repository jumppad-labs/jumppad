package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeRandomNumber is the resource for generating random numbers
const TypeRandomNumber string = "random_number"

// allows the generate of CA certificates
type RandomNumber struct {
	types.ResourceMetadata `hcl:",remain"`

	Minimum int `hcl:"minimum" json:"minimum"`
	Maximum int `hcl:"maximum" json:"maximum"`

	// Output parameters
	Value int `hcl:"value,optional" json:"value"`
}

func (c *RandomNumber) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*RandomNumber)
			c.Value = kstate.Value
		}
	}

	return nil
}
