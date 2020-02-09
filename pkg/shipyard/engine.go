package shipyard

import (
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
)

// Clients contains clients which are responsible for creating and destrying reources
type Clients struct {
	Docker         clients.Docker
	ContainerTasks clients.ContainerTasks
	Kubernetes     clients.Kubernetes
	HTTP           clients.HTTP
	Command        clients.Command
}

// Engine is responsible for creating and destroying resources
type Engine struct {
	providers [][]providers.Provider
	clients   *Clients
	config    *config.Config
	log       hclog.Logger
}

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l hclog.Logger) (*Clients, error) {
	dc, err := clients.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := clients.NewKubernetes(60 * time.Second)

	ec := clients.NewCommand(30*time.Second, l)

	ct := clients.NewDockerTasks(dc, l)

	hc := clients.NewHTTP(60*time.Second, l)

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Command:        ec,
		HTTP:           hc,
	}, nil
}

// NewWithFolder creates a new shipyard engine with a given configuration folder
func NewWithFolder(folder string, l hclog.Logger) (*Engine, error) {
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
	cl, err := GenerateClients(l)
	if err != nil {
		return nil, err
	}

	e := New(cc, cl, l)

	return e, nil
}

// New engine using the given configuration and clients
func New(c *config.Config, cc *Clients, l hclog.Logger) *Engine {
	p := generateProviders(c, cc, l)

	return &Engine{
		providers: p,
		clients:   cc,
		config:    c,
		log:       l,
	}
}

// Apply the current config creating the resources
func (e *Engine) Apply() error {
	// loop through each group
	for _, g := range e.providers {
		// apply the provider in parallel
		err := createParallel(g)
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

		err := destroyParallel(e.providers[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// ResourceCount defines the number of resources in a plan
func (e *Engine) ResourceCount() int {
	return e.config.ResourceCount()
}

// Blueprint returns the blueprint for the current config
func (e *Engine) Blueprint() *config.Blueprint {
	return e.config.Blueprint
}

// createParallel is just a quick implementation for now to test the UX
func createParallel(p []providers.Provider) error {
	errs := make(chan error)
	done := make(chan struct{})

	// create the wait group and set the size to the provider length
	wg := sync.WaitGroup{}
	wg.Add(len(p))

	for _, pr := range p {
		go func(pr providers.Provider) {
			err := pr.Create()
			if err != nil {
				errs <- err
			}

			wg.Done()
		}(pr)
	}

	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case err := <-errs:
		return err
	}

}

// destroyParallel is just a quick implementation for now to test the UX
func destroyParallel(p []providers.Provider) error {
	// create the wait group and set the size to the provider length
	wg := sync.WaitGroup{}
	wg.Add(len(p))

	for _, pr := range p {
		go func(pr providers.Provider) {
			pr.Destroy()
			wg.Done()
		}(pr)
	}

	wg.Wait()

	return nil
}

// generateProviders returns providers grouped together in order of execution
func generateProviders(c *config.Config, cc *Clients, l hclog.Logger) [][]providers.Provider {
	oc := make([][]providers.Provider, 7)
	oc[0] = make([]providers.Provider, 0)
	oc[1] = make([]providers.Provider, 0)
	oc[2] = make([]providers.Provider, 0)
	oc[3] = make([]providers.Provider, 0)
	oc[4] = make([]providers.Provider, 0)
	oc[5] = make([]providers.Provider, 0)
	oc[6] = make([]providers.Provider, 0)

	p := providers.NewNetwork(c.WAN, cc.Docker, l)
	oc[0] = append(oc[0], p)

	for _, n := range c.Networks {
		p := providers.NewNetwork(n, cc.Docker, l)
		oc[0] = append(oc[0], p)
	}

	for _, c := range c.Containers {
		p := providers.NewContainer(*c, cc.ContainerTasks, l)
		oc[1] = append(oc[1], p)
	}

	for _, c := range c.Clusters {
		p := providers.NewCluster(*c, cc.ContainerTasks, cc.Kubernetes, cc.HTTP, l)
		oc = append(oc, p)
	}

	for _, c := range c.HelmCharts {
		p := providers.NewHelm(c, cc.Kubernetes, l)
		oc[3] = append(oc[3], p)
	}

	for _, c := range c.K8sConfig {
		p := providers.NewK8sConfig(c, cc.Kubernetes, l)
		oc[4] = append(oc[4], p)
	}

	for _, c := range c.Ingresses {
		p := providers.NewIngress(*c, cc.ContainerTasks, l)
		oc[1] = append(oc[1], p)
	}

	if c.Docs != nil {
		p := providers.NewDocs(c.Docs, cc.ContainerTasks, l)
		oc[5] = append(oc[5], p)
	}

	for _, c := range c.LocalExecs {
		p := providers.NewLocalExec(c, cc.Command, l)
		oc[6] = append(oc[6], p)
	}

	for _, c := range c.RemoteExecs {
		p := providers.NewRemoteExec(*c, cc.ContainerTasks, l)
		oc[6] = append(oc[6], p)
	}

	return oc
}
