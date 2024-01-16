package config

import (
	"reflect"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// Provider defines an interface to be implemented by providers
//
//go:generate mockery --name Provider --filename provider.go
type Provider interface {
	// Init is called when the provider is created, it is passed a logger that
	// can be used for any logging purposes. Any other clients must be created
	// by the provider
	Init(types.Resource, sdk.Logger) error

	// Create is called when a resource does not exist or creation has previously
	// failed and 'up' is run
	Create() error

	// Destroy is called when a resource is failed or created and 'down' is run
	Destroy() error

	// Refresh is called when a resource is created and 'up' is run
	Refresh() error

	// Changed returns if a resource has changed since the last run
	Changed() (bool, error)

	// Lookup is a utility to determine the existence of a resource
	Lookup() ([]string, error)
}

// ConfigWrapper allows the provider config to be deserialized to a type
type ConfigWrapper struct {
	Type  string
	Value interface{}
}

type Providers interface {
	GetProvider(c types.Resource) Provider
}

type ProvidersImpl struct {
	clients *clients.Clients
}

func NewProviders(c *clients.Clients) Providers {
	return &ProvidersImpl{c}
}

func (p *ProvidersImpl) GetProvider(r types.Resource) Provider {
	// find the type
	if t, ok := registeredProviders[r.Metadata().ResourceType]; ok {
		ptr := reflect.New(reflect.TypeOf(t).Elem())

		prov := ptr.Interface().(Provider)
		prov.Init(r, p.clients.Logger)

		return prov
	}

	return nil
}
