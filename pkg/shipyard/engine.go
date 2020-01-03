package shipyard

import (
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
)

// Clients contains clients which are responsible for creating and destrying reources
type Clients struct {
	Docker     clients.Docker
	Kubernetes clients.Kubernetes
}

// Engine is responsible for creating and destroying resources
type Engine struct {
	providers []providers.Provider
	clients   *Clients
	config    *config.Config
}

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients() (*Clients, error) {
	dc, err := clients.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := clients.NewKubernetes()

	return &Clients{
		Docker:     dc,
		Kubernetes: kc,
	}, nil
}

// NewWithFolder creates a new shipyard engine with a given configuration folder
func NewWithFolder(folder string) (*Engine, error) {
	var err error
	cc, err := config.New()
	if err != nil {
		return nil, err
	}

	err = config.ParseFolder(folder, cc)
	if err != nil {
		return nil, err
	}

	err = config.ParseReferences(cc)
	if err != nil {
		return nil, err
	}

	// create providers
	cl, err := GenerateClients()
	if err != nil {
		return nil, err
	}

	e := New(cc, cl)

	return e, nil
}

// New engine using the given configuration and clients
func New(c *config.Config, cc *Clients) *Engine {
	p := generateProviders(c, cc)

	return &Engine{
		providers: p,
		clients:   cc,
		config:    c,
	}
}

// Apply the current config creating the resources
func (e *Engine) Apply() error {
	for _, p := range e.providers {
		err := p.Create()
		if err != nil {
			return err
		}
	}

	return nil
}

// Destroy the resources defined by the config
func (e *Engine) Destroy() error {
	// should run through the providers in reverse order
	// to ensure objects with dependencies are destroyed first
	for i := len(e.providers) - 1; i > -1; i-- {
		err := e.providers[i].Destroy()
		if err != nil {
			return err
		}
	}

	return nil
}

// Blueprint returns the blueprint for the current config
func (e *Engine) Blueprint() *config.Blueprint {
	return e.config.Blueprint
}

func generateProviders(c *config.Config, cc *Clients) []providers.Provider {
	oc := make([]providers.Provider, 0)

	p := providers.NewNetwork(c.WAN, cc.Docker)
	oc = append(oc, p)

	for _, n := range c.Networks {
		p := providers.NewNetwork(n, cc.Docker)
		oc = append(oc, p)
	}

	for _, c := range c.Containers {
		p := providers.NewContainer(c, cc.Docker)
		oc = append(oc, p)
	}

	for _, c := range c.Clusters {
		p := providers.NewCluster(c, cc.Docker, cc.Kubernetes)
		oc = append(oc, p)
	}

	for _, c := range c.HelmCharts {
		p := providers.NewHelm(c, cc.Kubernetes)
		oc = append(oc, p)
	}

	for _, c := range c.K8sConfig {
		p := providers.NewK8sConfig(c, cc.Kubernetes)
		oc = append(oc, p)
	}

	for _, c := range c.Ingresses {
		p := providers.NewIngress(c, cc.Docker)
		oc = append(oc, p)
	}

	if c.Docs != nil {
		p := providers.NewDocs(c.Docs, cc.Docker)
		oc = append(oc, p)
	}

	return oc
}
