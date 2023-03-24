package shipyard

import (

	// "fmt"

	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/shipyard/constants"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// Clients contains clients which are responsible for creating and destroying resources
type Clients struct {
	Docker         clients.Docker
	ContainerTasks clients.ContainerTasks
	Kubernetes     clients.Kubernetes
	Helm           clients.Helm
	HTTP           clients.HTTP
	Nomad          clients.Nomad
	Command        clients.Command
	Logger         hclog.Logger
	Getter         clients.Getter
	Browser        clients.System
	ImageLog       clients.ImageLog
	Connector      clients.Connector
	TarGz          *clients.TarGz
}

// Engine defines an interface for the Shipyard engine
type Engine interface {
	GetClients() *Clients
	Apply(string) ([]types.Resource, error)

	// ApplyWithVariables applies a configuration file or directory containing
	// configuration. Optionally the user can provide a map of variables which the configuration
	// uses and / or a file containing variables.
	ApplyWithVariables(path string, variables map[string]string, variablesFile string) ([]types.Resource, error)
	ParseConfig(string) error
	ParseConfigWithVariables(string, map[string]string, string) error
	Destroy(string, bool) error
	ResourceCount() int
	ResourceCountForType(string) int
	Blueprint() *resources.Blueprint
}

// EngineImpl is responsible for creating and destroying resources
type EngineImpl struct {
	clients     *Clients
	config      *hclconfig.Config
	log         hclog.Logger
	getProvider getProviderFunc
	sync        sync.Mutex
}

// defines a function which is used for generating providers
// enables the replacement in tests to inject mocks
type getProviderFunc func(c types.Resource, cl *Clients) providers.Provider

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l hclog.Logger) (*Clients, error) {
	dc, err := clients.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := clients.NewKubernetes(60*time.Second, l)

	hec := clients.NewHelm(l)

	ec := clients.NewCommand(30*time.Second, l)

	hc := clients.NewHTTP(1*time.Second, l)

	nc := clients.NewNomad(hc, 1*time.Second, l)

	bp := clients.NewGetter(false)

	bc := &clients.SystemImpl{}

	il := clients.NewImageFileLog(utils.ImageCacheLog())

	tgz := &clients.TarGz{}

	ct := clients.NewDockerTasks(dc, il, tgz, l)

	co := clients.DefaultConnectorOptions()
	cc := clients.NewConnector(co)

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Helm:           hec,
		Command:        ec,
		HTTP:           hc,
		Nomad:          nc,
		Logger:         l,
		Getter:         bp,
		Browser:        bc,
		ImageLog:       il,
		Connector:      cc,
		TarGz:          tgz,
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

// GetClients returns the clients from the engine
func (e *EngineImpl) GetClients() *Clients {
	return e.clients
}

func (e *EngineImpl) Blueprint() *resources.Blueprint {
	return nil
}

// ParseConfig parses the given Shipyard files and creating the resource types but does
// not apply or destroy the resources.
// This function can be used to check the validity of a configuration without making changes
func (e *EngineImpl) ParseConfig(path string) error {
	return e.ParseConfigWithVariables(path, nil, "")
}

// ParseConfigWithVariables parses the given Shipyard files and creating the resource types but does
// not apply or destroy the resources.
// This function can be used to check the validity of a configuration without making changes
func (e *EngineImpl) ParseConfigWithVariables(path string, vars map[string]string, variablesFile string) error {
	err := e.readAndProcessConfig(path, vars, variablesFile, e.createCallback)
	if err != nil {
		return err
	}

	return nil
}

// Apply the configuration and create or destroy the resources
func (e *EngineImpl) Apply(path string) ([]types.Resource, error) {
	return e.ApplyWithVariables(path, nil, "")
}

// ApplyWithVariables applies the current config creating the resources
func (e *EngineImpl) ApplyWithVariables(path string, vars map[string]string, variablesFile string) ([]types.Resource, error) {
	// abs paths
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	e.log.Info("Creating resources from configuration", "path", path)

	if variablesFile != "" {
		variablesFile, err = filepath.Abs(variablesFile)
		if err != nil {
			return nil, err
		}
	}

	err = e.readAndProcessConfig(path, vars, variablesFile, e.createCallback)
	if err != nil {
		return nil, err
	}

	//createdResource := []types.Resource{}

	//// walk the dag and apply the config
	//w := dag.Walker{}
	//w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
	//	r, ok := v.(types.Resource)

	//	// not a resource quit
	//	if !ok {
	//		return nil
	//	}

	//	// get the provider to create the resource
	//	p := e.getProvider(r, e.clients)

	//	if p == nil {
	//		r.Metadata().Status = config.Failed
	//		return diags.Append(fmt.Errorf("Unable to create provider for resource Name: %s, Type: %s", r.Info().Name, r.Info().Type))
	//	}

	//	switch r.Info().Status {
	//	// Normal case for PendingUpdate is do nothing
	//	// PendingModification causes a resource to be
	//	// destroyed before created
	//	case config.PendingModification:
	//		fallthrough

	//		// Always attempt to destroy and re-create failed resources
	//	case config.Failed:
	//		err = p.Destroy()
	//		if err != nil {
	//			r.Info().Status = config.Failed
	//			return diags.Append(err)
	//		}

	//		fallthrough // failed resources should always attempt recreation

	//	// Create new resources
	//	case config.PendingCreation:
	//		createErr := p.Create()
	//		if createErr != nil {
	//			r.Info().Status = config.Failed
	//			return diags.Append(createErr)
	//		}

	//	case config.PendingUpdate:
	//		// do nothing for pending updates

	//	case config.Disabled:
	//		// do nothing for disabled updates
	//	}

	//	// set the status only if not disabled
	//	if r.Info().Status != config.Disabled {
	//		r.Info().Status = config.Applied
	//	}

	//	appendResources(&createdResource, r)

	//	return nil
	//}

	//w.Update(d)
	//tf := w.Wait()
	//if tf.Err() != nil {
	//	err = tf.Err()
	//}

	//if len(e.config.Resources) > 0 {
	//	// save the state regardless of error
	//	jerr := e.config.ToJSON(utils.StatePath())
	//	if jerr != nil {
	//		return createdResource, jerr
	//	}

	//	return createdResource, err
	//}

	return nil, nil
}

// createCallback is used by hclconfig when attempting to create resources
func (e *EngineImpl) createCallback(r types.Resource) error {
	fqdn := types.FQDNFromResource(r)
	e.log.Info("Created resource", "fqdn", fqdn.String())

	p := e.getProvider(r, e.clients)

	if p == nil {
		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		return fmt.Errorf("unable to create provider for resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
	}

	switch r.Metadata().Properties[constants.PropertyStatus] {
	// Normal case for PendingUpdate is do nothing
	// PendingModification causes a resource to be
	// destroyed before created
	case constants.StatusTainted:
		fallthrough

		// Always attempt to destroy and re-create failed resources
	case constants.StatusFailed:
		err := p.Destroy()
		if err != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
			return err
		}

		fallthrough // failed resources should always attempt recreation

	case constants.StatusDisabled:
		// do nothing for disabled updates
		return nil

	default:
		createErr := p.Create()
		if createErr != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
			return createErr
		}
	}

	// set the status only if not disabled
	r.Metadata().Properties[constants.PropertyStatus] = constants.StatusCreated

	return nil
}

// Destroy the resources defined by the config
func (e *EngineImpl) Destroy(path string, allResources bool) error {
	//	d, err := e.readConfig(path, nil, "")
	//	if err != nil {
	//		return err
	//	}
	//
	//	// make sure we destroy everything
	//	if allResources {
	//		for _, i := range e.config.Resources {
	//			if i.Info().Status != config.Disabled {
	//				i.Info().Status = config.PendingUpdate
	//			}
	//		}
	//	}
	//
	//	// walk the dag and apply the config
	//	w := dag.Walker{}
	//	w.Reverse = true
	//	w.Callback = func(v dag.Vertex) (diags tfdiags.Diagnostics) {
	//		// check if the resource needs to be created and if so create
	//		if r, ok := v.(config.Resource); ok {
	//			switch r.Info().Status {
	//			case config.PendingUpdate:
	//				// do nothing for disabled resources
	//				if r.Info().Status == config.Disabled {
	//					r.Info().Status = config.Destroyed
	//					return nil
	//				}
	//
	//				// get the provider to create the resource
	//				p := e.getProvider(r, e.clients)
	//				if p == nil {
	//					r.Info().Status = config.Failed
	//					return diags.Append(fmt.Errorf("Unable to create provider for resource Name: %s, Type: %s", r.Info().Name, r.Info().Type))
	//				}
	//
	//				// execute
	//				destroyErr := p.Destroy()
	//				if destroyErr != nil {
	//					r.Info().Status = config.Failed
	//					return diags.Append(destroyErr)
	//				}
	//
	//				fallthrough
	//			case config.Disabled:
	//				// set the status
	//				r.Info().Status = config.Destroyed
	//			}
	//		}
	//
	//		return nil
	//	}
	//
	//	w.Update(d)
	//	tf := w.Wait()
	//	if tf.Err() != nil {
	//		err = tf.Err()
	//	}
	//
	//	// remove any destroyed nodes from the state
	//	cn := config.New()
	//	for _, i := range e.config.Resources {
	//		if i.Info().Status != config.Destroyed {
	//			cn.AddResource(i)
	//		}
	//	}
	//
	//	// save the state regardless of error
	//	if len(cn.Resources) > 0 {
	//		err = cn.ToJSON(utils.StatePath())
	//		if err != nil {
	//			return err
	//		}
	//	} else {
	//		// if no resources in the state delete
	//		os.RemoveAll(utils.StatePath())
	//	}

	//return tf.Err()
	return nil
}

// ResourceCount defines the number of resources in a plan
func (e *EngineImpl) ResourceCount() int {
	return e.config.ResourceCount()
}

// ResourceCountForType returns the count of resources matching the given type
func (e *EngineImpl) ResourceCountForType(t string) int {
	r, err := e.config.FindResourcesByType(t)
	if err != nil {
		return 0
	}

	return len(r)
}

func (e *EngineImpl) readAndProcessConfig(path string, variables map[string]string, variablesFile string, callback hclconfig.ProcessCallback) error {
	stateConfig := hclconfig.NewConfig()
	pathConfig := hclconfig.NewConfig()
	cache := &resources.ImageCache{
		ResourceMetadata: types.ResourceMetadata{
			Name: "default",
			Type: resources.TypeImageCache,
		},
	}

	// load the existing state if a state file exists
	if _, err := os.Stat(utils.StatePath()); err == nil {
		d, err := os.ReadFile(utils.StatePath())
		if err != nil {
			return fmt.Errorf("unable to read state file: %s", err)
		}

		// create a new parser and unmarshal the state from json
		hclParser := setupHCLConfig(nil, nil, nil)
		stateConfig, err = hclParser.UnmarshalJSON(d)
		if err != nil {
			return fmt.Errorf("unable to parsing state file: %s", err)
		}
	} else {
		e.log.Debug("State file does not exist")
	}

	// check to see we have an image cache
	caches, err := pathConfig.FindResourcesByType(resources.TypeImageCache)
	if err == nil && len(caches) == 1 {
		// add a default resource for the docker caching proxy
		cache = caches[0].(*resources.ImageCache)
	} else {
		// add the newly created cache to the state as one does not exist
		stateConfig.AppendResource(cache)
	}

	if path != "" {
		variablesFiles := []string{}
		if variablesFile != "" {
			variablesFiles = append(variablesFiles, variablesFile)
		}

		var err error
		hclParser := setupHCLConfig(callback, variables, variablesFiles)

		if utils.IsHCLFile(path) {
			// ParseFile processes the HCL, builds a graph of resources then calls
			// the callback for each resource in order
			pathConfig, err = hclParser.ParseFile(path)
			if err != nil {
				return err
			}
		} else {
			// ParseFolder processes the HCL, builds a graph of resources then calls
			// the callback for each resource in order
			pathConfig, err = hclParser.ParseDirectory(path)
			if err != nil {
				return err
			}
		}
	}

	// if there is state merge the state and items to be created or deleted
	if stateConfig != nil {
		stateConfig.AppendResourcesFromConfig(pathConfig)
	} else {
		stateConfig = pathConfig
	}

	// we now need to find all the networks defined in the config and
	// add them to the image cache's dependencies
	cache.DependsOn = []string{}
	networks, _ := stateConfig.FindResourcesByType(resources.TypeNetwork)
	for _, n := range networks {
		cache.DependsOn = append(cache.DependsOn, n.Metadata().ID)
	}

	// again we need to execute the provider for the cache to update any new networks
	p := e.getProvider(cache, e.GetClients())
	p.Create()

	// set the config
	e.config = stateConfig

	return nil
}
