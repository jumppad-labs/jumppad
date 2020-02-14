package shipyard

import (

	// "fmt"

	"log"
	"os"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"golang.org/x/xerrors"

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
	Logger         hclog.Logger
}

// Engine is responsible for creating and destroying resources
type Engine struct {
	clients     *Clients
	config      *config.Config
	log         hclog.Logger
	getProvider getProviderFunc
	sync        sync.Mutex
}

// defines a function which is used for generating providers
// enables the replacement in tests to inject mocks
type getProviderFunc func(c config.Resource, cl *Clients) providers.Provider

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
		Logger:         l,
	}, nil
}

// New creates a new shipyard engine
func New(l hclog.Logger) (*Engine, error) {
	var err error
	e := &Engine{}
	e.log = l
	e.getProvider = generateProviderImpl

	log.SetOutput(l.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Trace}))

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
	d, err := e.readConfig(path)
	if err != nil {
		return err
	}

	// walk the dag and apply the config
	w := dag.Walker{}
	w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// check if the resource needs to be created and if so create
		if r, ok := v.(config.Resource); ok && r.Info().Status == config.PendingCreation {
			// get the provider to create the resource
			p := e.getProvider(r, e.clients)
			if p == nil {
				r.Info().Status = config.Failed
				return diags.Append(err)
			}

			// execute
			err = p.Create()
			if err != nil {
				r.Info().Status = config.Failed
				return diags.Append(err)
			}

			// set the status
			r.Info().Status = config.Applied
		}

		return nil
	}

	w.Update(d)
	tf := w.Wait()
	if tf.Err() != nil {
		return tf.Err()
	}

	if len(e.config.Resources) > 0 {
		// save the state regardless of error
		err = e.config.ToJSON(utils.StatePath())
		if err != nil {
			return err
		}
	}

	return err
}

// Destroy the resources defined by the config
func (e *Engine) Destroy(path string, allResources bool) error {
	d, err := e.readConfig(path)
	if err != nil {
		return err
	}

	// make sure we destroy everything
	if allResources {
		for _, i := range e.config.Resources {
			i.Info().Status = config.PendingModification
		}
	}

	// walk the dag and apply the config
	w := dag.Walker{}
	w.Reverse = true
	w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// check if the resource needs to be created and if so create
		if r, ok := v.(config.Resource); ok && r.Info().Status == config.PendingModification {
			// get the provider to create the resource
			p := e.getProvider(r, e.clients)
			if p == nil {
				r.Info().Status = config.Failed
				return diags.Append(err)
			}

			// execute
			err = p.Destroy()
			if err != nil {
				r.Info().Status = config.Failed
				return diags.Append(err)
			}

			// set the status
			r.Info().Status = config.Destroyed
		}

		return nil
	}

	w.Update(d)
	tf := w.Wait()
	if tf.Err() != nil {
		return tf.Err()
	}

	// remove any destroyed nodes from the state
	cn := config.New()
	for _, i := range e.config.Resources {
		if i.Info().Status != config.Destroyed {
			cn.AddResource(i)
		}
	}

	// save the state regardless of error
	if len(cn.Resources) > 0 {
		err = cn.ToJSON(utils.StatePath())
		if err != nil {
			return err
		}
	} else {
		// if no resources in the state delete
		os.RemoveAll(utils.StatePath())
	}

	return err
}

func (e *Engine) readConfig(path string) (*dag.AcyclicGraph, error) {
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

		// if we are loading from files create the deps
		config.ParseReferences(cc)
	}

	// load the existing state
	sc := config.New()
	err := sc.FromJSON(utils.StatePath())
	if err != nil {
		// we do not have any state to create a new one
		e.log.Debug("Statefile does not exist")
	}

	// merge the state and items to be created or deleted
	sc.Merge(cc)

	// set the config
	e.config = sc

	// build a DAG
	d, err := e.config.DoYaLikeDAGs()
	if err != nil {
		return nil, xerrors.Errorf("Unable to create dependency graph: %w", err)
	}

	d.TransitiveReduction()

	err = d.Validate()
	if err != nil {
		return nil, xerrors.Errorf("Unable to validate dependency graph: %w", err)
	}

	return d, nil
}

// ResourceCount defines the number of resources in a plan
func (e *Engine) ResourceCount() int {
	return e.config.ResourceCount()
}

// Blueprint returns the blueprint for the current config
func (e *Engine) Blueprint() *config.Blueprint {
	return e.config.Blueprint
}

// generateProviderImpl returns providers grouped together in order of execution
func generateProviderImpl(c config.Resource, cc *Clients) providers.Provider {
	switch c.Info().Type {
	case config.TypeContainer:
		return providers.NewContainer(c.(*config.Container), cc.ContainerTasks, cc.Logger)
	case config.TypeDocs:
		return providers.NewDocs(c.(*config.Docs), cc.ContainerTasks, cc.Logger)
	case config.TypeExecRemote:
		return providers.NewRemoteExec(c.(*config.ExecRemote), cc.ContainerTasks, cc.Logger)
	case config.TypeExecLocal:
		return providers.NewExecLocal(c.(*config.ExecLocal), cc.Command, cc.Logger)
	case config.TypeHelm:
		return providers.NewHelm(c.(*config.Helm), cc.Kubernetes, cc.Helm, cc.Logger)
	case config.TypeIngress:
		return providers.NewIngress(c.(*config.Ingress), cc.ContainerTasks, cc.Logger)
	case config.TypeK8sCluster:
		return providers.NewK8sCluster(c.(*config.K8sCluster), cc.ContainerTasks, cc.Kubernetes, cc.HTTP, cc.Logger)
	case config.TypeK8sConfig:
		return providers.NewK8sConfig(c.(*config.K8sConfig), cc.Kubernetes, cc.Logger)
	case config.TypeNetwork:
		return providers.NewNetwork(c.(*config.Network), cc.Docker, cc.Logger)
	case config.TypeNomadCluster:
		return nil //providers.NewNomadCluster(c, cc.ContainerTasks, cc.Logger)
	}

	return nil
}
