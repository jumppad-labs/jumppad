package build

import "github.com/jumppad-labs/jumppad/pkg/config"

// register the types and provider
func init() {
	config.RegisterResource(TypeBuild, &Build{}, &Provider{})
}
