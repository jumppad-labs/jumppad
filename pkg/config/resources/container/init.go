package container

import "github.com/jumppad-labs/jumppad/pkg/config"

// register the types and provider
func init() {
	config.RegisterResource(TypeContainer, &Container{}, &Provider{})
	config.RegisterResource(TypeSidecar, &Sidecar{}, &Provider{})
}
