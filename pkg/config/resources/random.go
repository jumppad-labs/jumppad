package resources

import "github.com/jumppad-labs/hclconfig/types"

// TypeRandomNumber is the resource for generating random numbers
const TypeRandomNumber string = "random_number"

// allows the generation of random numbers
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
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*RandomID)
			c.Base64 = kstate.Base64
			c.Hex = kstate.Hex
			c.Dec = kstate.Dec
		}
	}

	return nil
}

// TypeRandomPassword is the resource for generating random passwords
const TypeRandomPassword string = "random_password"

// allows the generation of random Passwords
type RandomPassword struct {
	types.ResourceMetadata `hcl:",remain"`

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
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*RandomPassword)
			c.Value = kstate.Value
		}
	}

	return nil
}

// TypeRandomUUID is the resource for generating random UUIDs
const TypeRandomUUID string = "random_uuid"

// allows the generation of random UUIDs
type RandomUUID struct {
	types.ResourceMetadata `hcl:",remain"`

	// Output parameters
	Value string `hcl:"value,optional" json:"value"`
}

func (c *RandomUUID) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*RandomUUID)
			c.Value = kstate.Value
		}
	}

	return nil
}

// TypeRandomCreature is the resource for generating random creatures
const TypeRandomCreature string = "random_creature"

// allows the generation of random creatures
type RandomCreature struct {
	types.ResourceMetadata `hcl:",remain"`

	// Output parameters
	Value string `hcl:"value,optional" json:"value"`
}

func (c *RandomCreature) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*RandomCreature)
			c.Value = kstate.Value
		}
	}

	return nil
}

func boolPointer(value bool) *bool {
	return &value
}
