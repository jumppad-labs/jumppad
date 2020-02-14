package shipyard

import (

	// "fmt"

	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"

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
	clients     *Clients
	config      *config.Config
	log         hclog.Logger
	getProvider getProviderFunc
	stateLock   sync.Mutex
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
	e.getProvider = generateProvidersImpl

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
	d, err := e.readConfig(path, false)
	if err != nil {
		return err
	}

	// walk the dag and apply the config
	d.Walk(func(v dag.Vertex) tfdiags.Diagnostics {
		// check if the resource needs to be created and if so create

		// get the provider to create the resource

		// set the status
		return nil
	})

	// save the state regardless of error
	err = e.config.ToJSON(utils.StatePath())
	if err != nil {
		return err
	}

	return err
}

// Destroy the resources defined by the config
func (e *Engine) Destroy(path string, allResources bool) error {
	d, err := e.readConfig(path, false)
	if err != nil {
		return err
	}

	/*
		// walk the dag and apply the config
		r, err := d.Root()
		if err != nil {
			return err
		}

		d.ReverseDepthFirstWalk(r, func(v dag.Vertex) tfdiags.Diagnostics {
			// check if the resource needs to be created and if so create
			return nil
		})
	*/
	// save the state regardless of error
	err = e.config.ToJSON(utils.StatePath())
	if err != nil {
		return err
	}

	return err
}

func (e *Engine) readConfig(path string, delete bool) (*dag.AcyclicGraph, error) {
	// load the new config
	cc := config.New()
	if path != "" {
		if utils.IsHCLFile(path) {
			err := config.ParseHCLFile(path, cc)
			if err != nil {
				return nil, err
			}
		} else {
			err := config.ParseFolder(path, cc)
			if err != nil {
				return nil, err
			}
		}
	}

	// load the existing state
	sc := config.New()
	err := sc.FromJSON(utils.StatePath())
	if err != nil {
		return nil, err
	}

	// merge the state and items to be created or deleted
	sc.Merge(cc)

	// set the config
	e.config = sc

	// build a DAG
	return e.config.DoYaLikeDAGs()
}

// ResourceCount defines the number of resources in a plan
func (e *Engine) ResourceCount() int {
	return e.config.ResourceCount()
}

// Blueprint returns the blueprint for the current config
func (e *Engine) Blueprint() *config.Blueprint {
	return e.config.Blueprint
}

// generateProviders returns providers grouped together in order of execution
func generateProvidersImpl(c *config.Config, cc *Clients, l hclog.Logger) providers.Provider {

	return nil
}
