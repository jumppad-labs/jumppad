package config

// TypeModule is the resource string for a Module resource
const TypeModule ResourceType = "module"

// Module allows Shipyard configuration to be imported from external folder or
// GitHub repositories
type Module struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Source string `hcl:"source" json:"source"`

	Variables interface{} `hcl:"variables,optional" json:"variables,omitempty"`
}

// NewModule creates a new Module config resource
func NewModule(name string) *Module {
	return &Module{ResourceInfo: ResourceInfo{Name: name, Type: TypeModule, Status: PendingCreation}}
}
