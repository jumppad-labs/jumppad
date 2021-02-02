package config

const TypeVariable ResourceType = "variable"

// Output defines an output variable which can be set by a module
type Variable struct {
	ResourceInfo
	Default     interface{} `hcl:"default" json:"default"`                            // default value for a variable
	Description string      `hcl:"description,optional" json:"description,omitempty"` // description of the variable
}

// NewOutput creates a new output variable
func NewVariable(name string) *Variable {
	return &Variable{ResourceInfo: ResourceInfo{Name: name, Type: TypeVariable, Status: PendingCreation}}
}
