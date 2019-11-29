package shipyard

import (
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
	"github.com/shipyard-run/cli/providers"
)

type Clients struct {
	Docker clients.Docker
}

// Defines
type Engine struct {
	providers []providers.Provider
	clients   *Clients
}

func New(c *config.Config, cc *Clients) *Engine {
	p := generateProviders(c, cc)

	return &Engine{
		providers: p,
		clients:   cc,
	}
}

func (e *Engine) Apply() {
	for _, p := range e.providers {
		p.Create()
	}
}

func (e *Engine) Destroy() {

}

func generateProviders(c *config.Config, cc *Clients) []providers.Provider {
	oc := make([]providers.Provider, 0)

	for _, c := range c.Containers {
		p := providers.NewContainer(c, cc.Docker)
		oc = append(oc, p)
	}
	// first elements to create are networks
	/*
		for _, n := range c.Networks {
			p :=
			oc = append(oc, n)
		}

		for _, c := range c.Containers {
			oc = append(oc, c)
		}

		for _, c := range c.Clusters {
			oc = append(oc, c)
		}

		for _, c := range c.HelmCharts {
			oc = append(oc, c)
		}

		for _, c := range c.Ingresses {
			oc = append(oc, c)
		}
	*/

	return oc
}
