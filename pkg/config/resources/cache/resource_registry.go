package cache

import "github.com/jumppad-labs/hclconfig/types"

// RegistryAuth defines a structure for authenticating against a docker registry
type RegistryAuth struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Hostname string `hcl:"hostname,optional" json:"hostname"` // Optional hostname for authentication
	Username string `hcl:"username" json:"username"`          // Username for authentication, should not be an email
	Password string `hcl:"password" json:"password"`          // Password for authentication
}

// Registry defines a structure for registering additional registries
type Registry struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Hostname string        `hcl:"hostname" json:"hostname"`         // Hostname of the registry
	Auth     *RegistryAuth `hcl:"auth,block" json:"auth,omitempty"` // auth to authenticate against registry
}
