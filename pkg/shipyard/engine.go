package shipyard

import (

	// "fmt"

	"fmt"
	"io/ioutil"
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
//
//go:generate mockery --name Engine --filename engine.go
type Engine interface {
	GetClients() *Clients
	Apply(string) ([]types.Resource, error)

	// ApplyWithVariables applies a configuration file or directory containing
	// configuration. Optionally the user can provide a map of variables which the configuration
	// uses and / or a file containing variables.
	ApplyWithVariables(path string, variables map[string]string, variablesFile string) ([]types.Resource, error)
	ParseConfig(string) ([]types.Resource, error)
	ParseConfigWithVariables(string, map[string]string, string) ([]types.Resource, error)
	Destroy() error
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
func (e *EngineImpl) ParseConfig(path string) ([]types.Resource, error) {
	return e.ParseConfigWithVariables(path, nil, "")
}

// ParseConfigWithVariables parses the given Shipyard files and creating the resource types but does
// not apply or destroy the resources.
// This function can be used to check the validity of a configuration without making changes
func (e *EngineImpl) ParseConfigWithVariables(path string, vars map[string]string, variablesFile string) ([]types.Resource, error) {
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

	err = e.readAndProcessConfig(path, vars, variablesFile, func(r types.Resource) error {
		e.config.AppendResource(r)
		return nil
	})

	return e.config.Resources, err
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

	processErr := e.readAndProcessConfig(path, vars, variablesFile, e.createCallback)

	// we now need to find all the networks defined in the config and
	// add them to the image cache's dependencies
	// this will also ensure that the image cache is destroyed last
	cache := &resources.ImageCache{
		ResourceMetadata: types.ResourceMetadata{
			Name: "default",
			Type: resources.TypeImageCache,
		},
	}

	// check to see we have an image cache
	caches, err := e.config.FindResourcesByType(resources.TypeImageCache)
	if err == nil && len(caches) == 1 {
		// add a default resource for the docker caching proxy
		cache = caches[0].(*resources.ImageCache)
	} else {
		// add the newly created cache to the state as one does not exist
		e.config.AppendResource(cache)
	}

	cache.DependsOn = []string{}
	networks, _ := e.config.FindResourcesByType(resources.TypeNetwork)
	for _, n := range networks {
		cache.DependsOn = append(cache.DependsOn, n.Metadata().ID)
	}

	// again we need to execute the provider for the cache to update any new networks
	p := e.getProvider(cache, e.GetClients())
	err = p.Create()

	// save the state regardless of error
	stateErr := e.saveState()
	if stateErr != nil {
		e.log.Info("Unable to save state", "error", stateErr)
	}

	return e.config.Resources, processErr
}

// Destroy the resources defined by the state
func (e *EngineImpl) Destroy() error {
	e.log.Info("Destroying resources")

	// load the state
	err := e.loadState()
	if err != nil {
		e.log.Debug("State file does not exist")
	}

	// run through the graph and call the destroy callback
	// disabled resources are not included in this callback
	// image cache which is manually added by Apply process
	// should have the correct dependency graph to be
	// destroyed last
	err = e.config.Process(e.destroyCallback, true)
	if err != nil {
		// save the state
		stateErr := e.saveState()
		if stateErr != nil {
			// if we can not save the state, log
			e.log.Info("Unable to save state", "error", stateErr)
		}

		// return the process error
		return fmt.Errorf("error trying to call Destroy on provider: %s", err)
	}

	// remove the state
	return os.Remove(utils.StatePath())
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

	// load the state
	e.loadState()

	var parseError error
	var parsedConfig *hclconfig.Config

	if path != "" {
		variablesFiles := []string{}
		if variablesFile != "" {
			variablesFiles = append(variablesFiles, variablesFile)
		}

		hclParser := setupHCLConfig(callback, variables, variablesFiles)

		if utils.IsHCLFile(path) {
			// ParseFile processes the HCL, builds a graph of resources then calls
			// the callback for each resource in order
			//
			// We are not using the returned config as the resources are added to the
			// state on the callback
			//
			// If the callback returns an error we need to save the state and exit
			parsedConfig, parseError = hclParser.ParseFile(path)
			if parseError != nil {
				parseError = fmt.Errorf("error parsing file %s, error: %s", path, parseError)
			}
		} else {
			// ParseFolder processes the HCL, builds a graph of resources then calls
			// the callback for each resource in order
			//
			// We are not using the returned config as the resources are added to the
			// state on the callback
			//
			// If the callback returns an error we need to save the state and exit
			parsedConfig, parseError = hclParser.ParseDirectory(path)
			if parseError != nil {
				parseError = fmt.Errorf("error parsing directory %s, error: %s", path, parseError)
			}
		}

		// process is not called for disabled resources, add manually
		err := e.appendDisabledResources(parsedConfig)
		if err != nil {
			return fmt.Errorf("error parsing directory %s, error: %s", path, parseError)
		}
	}

	return parseError
}

// appends disabled resources in the given config to the engines config
func (e *EngineImpl) appendDisabledResources(c *hclconfig.Config) error {
	for _, r := range c.Resources {
		if r.Metadata().Disabled {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusDisabled

			// if the resource already exists remove it
			er, err := e.config.FindResource(types.FQDNFromResource(r).String())
			if err == nil {
				e.config.RemoveResource(er)
			}

			// add the resource to the state
			err = e.config.AppendResource(r)
			if err != nil {
				return fmt.Errorf("unable to add disabled resource: %s", err)
			}
		}
	}

	return nil
}

func (e *EngineImpl) createCallback(r types.Resource) error {
	p := e.getProvider(r, e.clients)

	if p == nil {
		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		return fmt.Errorf("unable to create provider for resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
	}

	// we need to check if a resource exists in the state
	// if so the status should take precedence as all new
	// resources will have an empty state
	sr, err := e.config.FindResource(r.Metadata().ID)
	if err == nil {
		r.Metadata().Properties[constants.PropertyStatus] = sr.Metadata().Properties[constants.PropertyStatus]

		// remove the resource, we will add the new version to the state
		e.config.RemoveResource(r)
	}

	var providerError error
	switch r.Metadata().Properties[constants.PropertyStatus] {
	// Normal case for PendingUpdate is do nothing
	// PendingModification causes a resource to be
	// destroyed before created
	case constants.StatusTainted:
		fallthrough

	// Always attempt to destroy and re-create failed resources
	case constants.StatusFailed:
		providerError = p.Destroy()
		if providerError != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		}

		fallthrough // failed resources should always attempt recreation

	default:
		providerError = p.Create()
		if providerError != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		} else {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusCreated
		}
	}

	// add the resource to the state
	e.config.AppendResource(r)

	return providerError
}

func (e *EngineImpl) destroyCallback(r types.Resource) error {
	fqdn := types.FQDNFromResource(r)

	// do nothing for disabled resources
	if r.Metadata().Disabled {
		e.log.Info("Skipping disabled resource", "fqdn", fqdn.String())

		e.config.RemoveResource(r)
		return nil
	}

	p := e.getProvider(r, e.clients)

	if p == nil {
		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		return fmt.Errorf("unable to create provider for resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
	}

	err := p.Destroy()
	if err != nil {
		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		return fmt.Errorf("unable to destroy resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
	}

	// remove from the state only if not errored
	e.config.RemoveResource(r)

	return nil
}

func (e *EngineImpl) loadState() error {
	d, err := ioutil.ReadFile(utils.StatePath())
	if err != nil {
		e.config = hclconfig.NewConfig()
		return fmt.Errorf("unable to read state file: %s", err)
	}

	p := setupHCLConfig(nil, nil, nil)
	c, err := p.UnmarshalJSON(d)
	if err != nil {
		e.config = hclconfig.NewConfig()
		return fmt.Errorf("unable to unmarshal state file: %s", err)
	}

	e.config = c

	return nil
}

func (e *EngineImpl) saveState() error {
	// save the state regardless of error
	d, err := e.config.ToJSON()
	if err != nil {
		return fmt.Errorf("unable to serialize config to JSON: %s", err)
	}

	err = os.MkdirAll(utils.StateDir(), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory for state file '%s', error: %s", utils.StateDir(), err)
	}

	err = ioutil.WriteFile(utils.StatePath(), d, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to write state file '%s', error: %s", utils.StatePath(), err)
	}

	return nil
}
