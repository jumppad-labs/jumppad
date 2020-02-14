package shipyard

import (

	// "fmt"

	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	// "github.com/mitchellh/mapstructure"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// Clients contains clients which are responsible for creating and destrying reources
type Clients struct {
	Docker         clients.Docker
	ContainerTasks clients.ContainerTasks
	Kubernetes     clients.Kubernetes
	Helm           clients.Helm
	HTTP           clients.HTTP
	Command        clients.Command
}

// Engine is responsible for creating and destroying resources
type Engine struct {
	providers   [][]providers.Provider
	clients     *Clients
	config      *config.Config
	log         hclog.Logger
	getProvider getProviderFunc
	stateLock   sync.Mutex
	state       []providers.ConfigWrapper
}

// defines a function which is used for generating providers
// enables the replacement in tests to inject mocks
type getProviderFunc func(c *config.Config, cl *Clients, l hclog.Logger) providers.Provider

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l hclog.Logger) (*Clients, error) {
	dc, err := clients.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := clients.NewKubernetes(60 * time.Second)

	hec := clients.NewHelm(l)

	ec := clients.NewCommand(30*time.Second, l)

	ct := clients.NewDockerTasks(dc, l)

	hc := clients.NewHTTP(1*time.Second, l)

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Helm:           hec,
		Command:        ec,
		HTTP:           hc,
	}, nil
}

// New creates a new shipyard engine
func New(l hclog.Logger) (*Engine, error) {
	var err error
	e := &Engine{}
	e.log = l
	e.generateProviders = generateProvidersImpl

	// create the clients
	cl, err := GenerateClients(l)
	if err != nil {
		return nil, err
	}

	e.clients = cl

	return e, nil
}

// Apply the current config creating the resources
func (e *Engine) Apply(path string) error {
	err := e.readConfig(path, false)

	// loop through each group and apply
	for _, g := range e.providers {
		// apply the provider in parallel
		createErr := e.createParallel(g)
		if createErr != nil {
			err = createErr
			break
		}
	}

	// save the state regardless of error
	e.saveState()

	return err
}

// Destroy the resources defined by the config
func (e *Engine) Destroy(path string, allResources bool) error {
	err := e.readConfig(path, true)
	if err != nil {
		return err
	}

	// should run through the providers in reverse order
	// to ensure objects with dependencies are destroyed first
	for i := len(e.providers) - 1; i > -1; i-- {
		err := e.destroyParallel(e.providers[i])
		if err != nil {
			e.log.Error("Error destroying resource", "error", err)
		}
	}

	e.saveState()

	return nil
}

func (e *Engine) readConfig(path string, delete bool) error {
	// load the new config
	cc := config.New()
	if path != "" {
		if utils.IsHCLFile(path) {
			err := config.ParseHCLFile(path, cc)
			if err != nil {
				return err
			}
		} else {
			err := config.ParseFolder(path, cc)
			if err != nil {
				return err
			}
		}
	}

	// load the existing state
	sc, err := e.configFromState(utils.StatePath())
	if err != nil {
		return err
	}

	// merge the state and items to be created or deleted
	e.config = e.mergeConfigItems(cc, sc, delete)

	// parse the references for the config links
	err = config.ParseReferences(e.config)
	if err != nil {
		return err
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
func (e *Engine) createParallel(p []providers.Provider) error {
	// errs := make(chan error)
	// done := make(chan struct{})

	// // create the wait group and set the size to the provider length
	// wg := sync.WaitGroup{}
	// wg.Add(len(p))

	// for _, pr := range p {
	// 	go func(pr providers.Provider) {
	// 		defer wg.Done()

	// 		// only attempt to create if the state is awaiting creation
	// 		if pr.State() == config.PendingCreation {
	// 			err := pr.Create()
	// 			if err != nil {
	// 				errs <- err
	// 				return
	// 			}
	// 		}

	// 		// if an error happens then the state will end up incomplete
	// 		pr.SetState(config.Applied)

	// 		// append the state
	// 		e.stateLock.Lock()
	// 		defer e.stateLock.Unlock()
	// 		e.state = append(e.state, pr.Config())
	// 	}(pr)
	// }

	// go func() {
	// 	wg.Wait()
	// 	done <- struct{}{}
	// }()

	// select {
	// case <-done:
	// 	return nil
	// case err := <-errs:
	// 	return err
	// }
	return nil
}

// destroyParallel is just a quick implementation for now to test the UX
func (e *Engine) destroyParallel(p []providers.Provider) error {
	// create the wait group and set the size to the provider length
	// wg := sync.WaitGroup{}
	// wg.Add(len(p))

	// for _, pr := range p {
	// 	go func(pr providers.Provider) {
	// 		defer wg.Done()

	// 		if pr.State() == config.PendingModification {
	// 			pr.Destroy()
	// 			return
	// 		}

	// 		// only add to the state if we did not delete
	// 		e.stateLock.Lock()
	// 		defer e.stateLock.Unlock()
	// 		e.state = append(e.state, pr.Config())
	// 	}(pr)
	// }

	// wg.Wait()

	return nil
}

// generateProviders returns providers grouped together in order of execution
func generateProvidersImpl(c *config.Config, cc *Clients, l hclog.Logger) [][]providers.Provider {
	oc := make([][]providers.Provider, 7)
	oc[0] = make([]providers.Provider, 0)
	oc[1] = make([]providers.Provider, 0)
	oc[2] = make([]providers.Provider, 0)
	oc[3] = make([]providers.Provider, 0)
	oc[4] = make([]providers.Provider, 0)
	oc[5] = make([]providers.Provider, 0)
	oc[6] = make([]providers.Provider, 0)

	// if c.WAN != nil {
	// 	p := providers.NewNetwork(c.WAN, cc.Docker, l)
	// 	oc[0] = append(oc[0], p)
	// }

	// for _, n := range c.Networks {
	// 	p := providers.NewNetwork(n, cc.Docker, l)
	// 	oc[0] = append(oc[0], p)
	// }

	// for _, c := range c.Containers {
	// 	p := providers.NewContainer(*c, cc.ContainerTasks, l)
	// 	oc[1] = append(oc[1], p)
	// }

	// for _, c := range c.Ingresses {
	// 	p := providers.NewIngress(*c, cc.ContainerTasks, l)
	// 	oc[1] = append(oc[1], p)
	// }

	// if c.Docs != nil {
	// 	p := providers.NewDocs(c.Docs, cc.ContainerTasks, l)
	// 	oc[1] = append(oc[1], p)
	// }

	// for _, c := range c.Clusters {
	// 	p := providers.NewCluster(*c, cc.ContainerTasks, cc.Kubernetes, cc.HTTP, l)
	// 	oc[2] = append(oc[2], p)
	// }

	// for _, c := range c.HelmCharts {
	// 	p := providers.NewHelm(c, cc.Kubernetes, cc.Helm, l)
	// 	oc[3] = append(oc[3], p)
	// }

	// for _, c := range c.K8sConfig {
	// 	p := providers.NewK8sConfig(c, cc.Kubernetes, l)
	// 	oc[4] = append(oc[4], p)
	// }

	// for _, c := range c.LocalExecs {
	// 	p := providers.NewLocalExec(c, cc.Command, l)
	// 	oc[6] = append(oc[6], p)
	// }

	// for _, c := range c.RemoteExecs {
	// 	p := providers.NewRemoteExec(*c, cc.ContainerTasks, l)
	// 	oc[6] = append(oc[6], p)
	// }

	return oc
}
