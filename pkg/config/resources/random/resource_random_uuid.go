package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomUUID is the resource for generating random UUIDs
const TypeRandomUUID string = "random_uuid"

/*
The `random_uuid` resource allows the creation of random UUIDs.

@example
```
resource "random_uuid" "uuid" {}

	output "uuid" {
	    value = resource.random_uuid.uuid.value
	}

```

@resource
*/
type RandomUUID struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`
	/*
		The generated random UUID.

		@computed
	*/
	Value string `hcl:"value,optional" json:"value"`
}

func (c *RandomUUID) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*RandomUUID)
			c.Value = state.Value
		}
	}

	return nil
}
