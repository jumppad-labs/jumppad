package shipyard

import (

	// "fmt"

	"fmt"
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
	Nomad          clients.Nomad
	Command        clients.Command
	Logger         hclog.Logger
	Blueprints     clients.Blueprints
	Browser        clients.System
}

// Engine defines an interface for the Shipyard engine
type Engine interface {
	Apply(string) ([]config.Resource, error)
	Destroy(string, bool) error
	ResourceCount() int
	Blueprint() *config.Blueprint
}

// EngineImpl is responsible for creating and destroying resources
type EngineImpl struct {
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

	kc := clients.NewKubernetes(60*time.Second, l)

	hec := clients.NewHelm(l)

	ec := clients.NewCommand(30*time.Second, l)

	ct := clients.NewDockerTasks(dc, l)

	hc := clients.NewHTTP(1*time.Second, l)

	nc := clients.NewNomad(hc, 1*time.Second, l)

	bp := &clients.BlueprintsImpl{}

	bc := &clients.SystemImpl{}

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Helm:           hec,
		Command:        ec,
		HTTP:           hc,
		Nomad:          nc,
		Logger:         l,
		Blueprints:     bp,
		Browser:        bc,
	}, nil
}

// New creates a new shipyard engine
func New(l hclog.Logger) (Engine, error) {
	var err error
	e := &EngineImpl{}
	e.log = l
	e.getProvider = generateProviderImpl

	// Set the standard writer to our logger as the DAG uses the standard library log.
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
func (e *EngineImpl) Apply(path string) ([]config.Resource, error) {
	d, err := e.readConfig(path)
	if err != nil {
		return nil, err
	}

	createdResource := []config.Resource{}

	// walk the dag and apply the config
	w := dag.Walker{}
	w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// check if the resource needs to be created and if so create
		if r, ok := v.(config.Resource); ok &&
			(r.Info().Status == config.PendingCreation ||
				r.Info().Status == config.PendingModification ||
				r.Info().Status == config.Failed) {

			// get the provider to create the resource
			p := e.getProvider(r, e.clients)
			if p == nil {
				r.Info().Status = config.Failed
				return diags.Append(fmt.Errorf("Unable to create provider for resource Name: %s, Type: %s", r.Info().Name, r.Info().Type))
			}

			// if we are pending modification or failed try remove the old instance and
			// create again
			if r.Info().Status == config.PendingModification || r.Info().Status == config.Failed {
				err = p.Destroy()
				if err != nil {
					r.Info().Status = config.Failed
					return diags.Append(err)
				}
			}

			// create the resource
			err = p.Create()
			if err != nil {
				r.Info().Status = config.Failed
				return diags.Append(err)
			}

			// set the status
			r.Info().Status = config.Applied
			createdResource = append(createdResource, r)
		}

		return nil
	}

	w.Update(d)
	tf := w.Wait()
	if tf.Err() != nil {
		err = tf.Err()
	}

	// update the status of anything which is pending update as this
	// is not currently implemented
	// eventually we should compare resources and update as required
	for _, i := range e.config.Resources {
		if i.Info().Status == config.PendingUpdate {
			i.Info().Status = config.Applied
		}
	}

	if len(e.config.Resources) > 0 {
		// save the state regardless of error
		jerr := e.config.ToJSON(utils.StatePath())
		if jerr != nil {
			return createdResource, jerr
		}

		return createdResource, err
	}

	return nil, tf.Err()
}

// Destroy the resources defined by the config
func (e *EngineImpl) Destroy(path string, allResources bool) error {
	d, err := e.readConfig(path)
	if err != nil {
		return err
	}

	// make sure we destroy everything
	if allResources {
		for _, i := range e.config.Resources {
			i.Info().Status = config.PendingUpdate
		}
	}

	// walk the dag and apply the config
	w := dag.Walker{}
	w.Reverse = true
	w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// check if the resource needs to be created and if so create
		if r, ok := v.(config.Resource); ok && r.Info().Status == config.PendingUpdate {
			// get the provider to create the resource
			p := e.getProvider(r, e.clients)
			if p == nil {
				r.Info().Status = config.Failed
				return diags.Append(fmt.Errorf("Unable to create provider for resource Name: %s, Type: %s", r.Info().Name, r.Info().Type))
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
		err = tf.Err()
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

	return tf.Err()
}

// ResourceCount defines the number of resources in a plan
func (e *EngineImpl) ResourceCount() int {
	return e.config.ResourceCount()
}

// Blueprint returns the blueprint for the current config
func (e *EngineImpl) Blueprint() *config.Blueprint {
	return e.config.Blueprint
}

func (e *EngineImpl) readConfig(path string) (*dag.AcyclicGraph, error) {
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

// generateProviderImpl returns providers grouped together in order of execution
func generateProviderImpl(c config.Resource, cc *Clients) providers.Provider {
	switch c.Info().Type {
	case config.TypeContainer:
		return providers.NewContainer(c.(*config.Container), cc.ContainerTasks, cc.HTTP, cc.Logger)
	case config.TypeContainerIngress:
		return providers.NewContainerIngress(c.(*config.ContainerIngress), cc.ContainerTasks, cc.Logger)
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
	case config.TypeK8sIngress:
		return providers.NewK8sIngress(c.(*config.K8sIngress), cc.ContainerTasks, cc.Logger)
	case config.TypeNomadCluster:
		return providers.NewNomadCluster(c.(*config.NomadCluster), cc.ContainerTasks, cc.Nomad, cc.Logger)
	case config.TypeNomadIngress:
		return providers.NewNomadIngress(c.(*config.NomadIngress), cc.ContainerTasks, cc.Logger)
	case config.TypeNomadJob:
		return providers.NewNomadJob(c.(*config.NomadJob), cc.Nomad, cc.Logger)
	case config.TypeNetwork:
		return providers.NewNetwork(c.(*config.Network), cc.Docker, cc.Logger)
	}

	return nil
}
