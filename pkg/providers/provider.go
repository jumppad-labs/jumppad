package providers

import "github.com/shipyard-run/shipyard/pkg/config"

// Provider defines an interface to be implemented by providers
type Provider interface {
	Create() error
	Destroy() error
	Lookup() ([]string, error)
	Config() ConfigWrapper
	State() config.State
	SetState(state config.State)
}

// ConfigWrapper alows the provider config to be deserialized to a type
type ConfigWrapper struct {
	Type  string
	Value interface{}
}
