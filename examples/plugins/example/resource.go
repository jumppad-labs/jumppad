package example

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const Type string = "example"

type Example struct {
	types.ResourceBase `hcl:",remain"`

	Value string `hcl:"value" json:"value"`
}

func (c *Example) Process() error {
	cfg, err := LoadState()
	if err == nil {
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*Example)
			c.Value = state.Value
		}
	}

	return nil
}
