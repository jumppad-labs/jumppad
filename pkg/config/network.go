package config

// TypeNetwork is the string resource type for Network resources
const TypeNetwork ResourceType = "network"

// Network defines a Docker network
type Network struct {
	ResourceInfo `mapstructure:",squash"`

	Subnet string `hcl:"subnet" json:"subnet"`
}

// NewNetwork creates a new Network resource with the correct defaults
func NewNetwork(name string) *Network {
	return &Network{ResourceInfo: ResourceInfo{Name: name, Type: TypeNetwork, Status: PendingCreation}}
}
