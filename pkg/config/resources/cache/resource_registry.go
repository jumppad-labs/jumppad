package cache

import "github.com/jumppad-labs/hclconfig/types"

const TypeRegistry string = "container_registry"

/*
Registry defines a structure for registering additional registries for the image cache

@resource
*/
type Registry struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	// Hostname of the registry
	Hostname string `hcl:"hostname" json:"hostname"`
	// auth to authenticate against registry
	Auth *RegistryAuth `hcl:"auth,block" json:"auth,omitempty"`
}

// RegistryAuth defines a structure for authenticating against a docker registry
type RegistryAuth struct {
	// Hostname for authentication, can be different from registry hostname
	Hostname string `hcl:"hostname,optional" json:"hostname,omitempty"`
	// Username for authentication
	Username string `hcl:"username" json:"username"`
	// Password for authentication
	Password string `hcl:"password" json:"password"`
}
