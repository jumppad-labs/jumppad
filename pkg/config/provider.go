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
	sdk.Provider
}

// ConfigWrapper allows the provider config to be deserialized to a type
type ConfigWrapper struct {
	Type  string
	Value interface{}
}

type Providers interface {
	GetProvider(c types.Resource) sdk.Provider
}

type ProvidersImpl struct {
	clients *clients.Clients
}

func NewProviders(c *clients.Clients) Providers {
	return &ProvidersImpl{c}
}

func (p *ProvidersImpl) GetProvider(r types.Resource) sdk.Provider {
	// find the type
	if t, ok := registeredProviders[r.Metadata().Type]; ok {
		ptr := reflect.New(reflect.TypeOf(t).Elem())

		prov := ptr.Interface().(Provider)
		prov.Init(r, p.clients.Logger)

		return prov
	}

	return nil
}
