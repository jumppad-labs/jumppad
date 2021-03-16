package config

const TypeOutput ResourceType = "output"

// Output defines an output variable which can be set by a module
type Output struct {
	ResourceInfo `mapstructure:",squash"`

	Value string `hcl:"value,optional" json:"value,omitempty"` // command to use when starting the container
}

// NewOutput creates a new output variable
func NewOutput(name string) *Output {
	return &Output{ResourceInfo: ResourceInfo{Name: name, Type: TypeOutput, Status: PendingCreation}}
}
