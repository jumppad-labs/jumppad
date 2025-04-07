package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomNumber is the resource for generating random numbers
const TypeRandomNumber string = "random_number"

/*
The `random_number` resource allows the creation of random numbers.

```hcl

	resource "random_number" "name" {
	  ...
	}

```

@example
```

	resource "random_number" "port" {
	  minimum = 10000
	  maximum = 20000
	}

	output "random_number" {
	  value = resource.random_number.port.value
	}

```

@resource
*/
type RandomNumber struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		The minimum number to generate.

		```hcl
		minimum = 1000
		```
	*/
	Minimum int `hcl:"minimum" json:"minimum"`
	/*
		The maximum number to generate.

		```hcl
		maximum = 2000
		```
	*/
	Maximum int `hcl:"maximum" json:"maximum"`
	/*
		The generated random number.

		@computed
	*/
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
