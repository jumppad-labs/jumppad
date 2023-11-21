package cache

import "github.com/jumppad-labs/hclconfig/types"

const TypeRegistry string = "container_registry"

// Registry defines a structure for registering additional registries for the image cache
type Registry struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Hostname string        `hcl:"hostname" json:"hostname"`         // Hostname of the registry
	Auth     *RegistryAuth `hcl:"auth,block" json:"auth,omitempty"` // auth to authenticate against registry
}

// RegistryAuth defines a structure for authenticating against a docker registry
type RegistryAuth struct {
	Username string `hcl:"username" json:"username"` // Username for authentication, should not be an email
	Password string `hcl:"password" json:"password"` // Password for authentication
}
