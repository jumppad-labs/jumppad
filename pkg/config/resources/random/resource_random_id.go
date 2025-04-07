package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomID is the resource for generating random IDs
const TypeRandomID string = "random_id"

/*
The `random_id` resource allows the creation of random IDs.

```hcl

	resource "random_id" "name" {
	  ...
	}

```

@example
```

	resource "random_id" "id" {
	    byte_length = 4
	}

	output "id_base64" {
	    value = resource.random_id.meta.id.base64
	}

	output "id_hex" {
	    value = resource.random_id.meta.id.hex
	}

	output "id_dec" {
	    value = resource.random_id.meta.id.dec
	}

```

@resource
*/
type RandomID struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		The number of random bytes to produce. The minimum value is 1, which produces eight bits of randomness.

		```hcl
		byte_length = 4
		```
	*/
	ByteLength int64 `hcl:"byte_length" json:"byte_length"`
	/*
		The generated ID presented in base64.

		@computed
	*/
	Base64 string `hcl:"base64,optional" json:"base64"`
	/*
		The generated ID presented in padded hexadecimal digits.
		This result will always be twice as long as the requested byte length.

		@computed
	*/
	Hex string `hcl:"hex,optional" json:"hex"`
	/*
		The generated ID presented in non-padded decimal digits.

		@computed
	*/
	Dec string `hcl:"dec,optional" json:"dec"`
}

func (c *RandomID) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			state := r.(*RandomID)
			c.Base64 = state.Base64
			c.Hex = state.Hex
			c.Dec = state.Dec
		}
	}

	return nil
}
