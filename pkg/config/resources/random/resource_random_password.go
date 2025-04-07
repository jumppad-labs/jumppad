package random

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeRandomPassword is the resource for generating random passwords
const TypeRandomPassword string = "random_password"

/*
The `random_password` resource allows the creation of random passwords.

```hcl

	resource "random_password" "name" {
	  ...
	}

```

@example
```

	resource "random_password" "password" {
	    length = 32
	}

	output "password" {
	    value = resource.random_password.password.value
	}

```

@resource
*/
type RandomPassword struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		The length of the string desired.
		The minimum value for length is 1 and, length must also be >= (`min_upper` + `min_lower` + `min_numeric` + `min_special`).

		```hcl
		length = 32
		```
	*/
	Length int64 `hcl:"length" json:"lenght"`
	/*
		Supply your own list of special characters to use for string generation.
		This overrides the default character list in the special argument.
		The special argument must still be set to `true` for any overwritten characters to be used in generation.

		```hcl
		override_special = "!*()"
		```
	*/
	OverrideSpecial string `hcl:"override_special,optional" json:"override_special"`
	/*
		Include special characters in the result. These are `!@#$%&*()-_=+[]{}<>:?`.

		```hcl
		special = true
		```
	*/
	Special *bool `hcl:"special,optional" json:"special" default:"true"`
	/*
		Include numeric characters in the result.

		```hcl
		numeric = true
		```
	*/
	Numeric *bool `hcl:"numeric,optional" json:"numeric" default:"true"`
	/*
		Include lowercase alphabet characters in the result.

		```hcl
		lower = true
		```
	*/
	Lower *bool `hcl:"lower,optional" json:"lower" default:"true"`
	/*
		Include uppercase alphabet characters in the result.

		```hcl
		upper = true
		```
	*/
	Upper *bool `hcl:"upper,optional" json:"upper" default:"true"`
	/*
		Minimum number of special characters in the result.

		```hcl
		min_special = 4
		```
	*/
	MinSpecial int64 `hcl:"min_special,optional" json:"min_special" default:"0"`
	/*
		Minimum number of numeric characters in the result.

		```hcl
		min_numeric = 4
		```
	*/
	MinNumeric int64 `hcl:"min_numeric,optional" json:"min_numeric" default:"0"`
	/*
		Minimum number of lowercase alphabet characters in the result.

		```hcl
		min_lower = 4
		```
	*/
	MinLower int64 `hcl:"min_lower,optional" json:"min_lower" default:"0"`
	/*
		Minimum number of uppercase alphabet characters in the result.

		```hcl
		min_upper = 4
		```
	*/
	MinUpper int64 `hcl:"min_upper,optional" json:"min_upper" default:"0"`
	/*
		The generated random password.

		@computed
	*/
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
