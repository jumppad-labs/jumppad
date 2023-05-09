package shipyard

import (

	// "fmt"

	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/providers"
	"github.com/jumppad-labs/jumppad/pkg/shipyard/constants"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/hclconfig/types"
)

// Clients contains clients which are responsible for creating and destroying resources

// Engine defines an interface for the Shipyard engine
//
//go:generate mockery --name Engine --filename engine.go
type Engine interface {
	GetClients() *clients.Clients
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
	clients     *clients.Clients
	config      *hclconfig.Config
	log         hclog.Logger
	getProvider getProviderFunc
}

// defines a function which is used for generating providers
// enables the replacement in tests to inject mocks
type getProviderFunc func(c types.Resource, cl *clients.Clients) providers.Provider

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l hclog.Logger) (*clients.Clients, error) {
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

	return &clients.Clients{
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
func (e *EngineImpl) GetClients() *clients.Clients {
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

	e.log.Info("Parsing configuration", "path", path)

	if variablesFile != "" {
		variablesFile, err = filepath.Abs(variablesFile)
		if err != nil {
			return nil, err
		}
	}

	// load the state
	c, err := resources.LoadState()
	if err != nil {
		e.log.Debug("unable to load state", "error", err)
	}

	e.config = c

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

	// load the state
	c, err := resources.LoadState()
	if err != nil {
		e.log.Debug("unable to load state", "error", err)
	}
	e.config = c

	// check to see we already have an image cache
	_, err = e.config.FindResourcesByType(resources.TypeImageCache)
	if err != nil {
		cache := &resources.ImageCache{
			ResourceMetadata: types.ResourceMetadata{
				Name:       "default",
				Type:       resources.TypeImageCache,
				ID:         "resource.image_cache.default",
				Properties: map[string]interface{}{},
			},
		}

		e.log.Debug("Creating new Image Cache", "id", cache.ID)

		p := e.getProvider(cache, e.clients)
		if p == nil {
			// this should never happen
			panic(err)
		}

		// create the cache
		err := p.Create()
		if err != nil {
			return nil, fmt.Errorf("unable to create image cache: %s", err)
		}

		cache.Properties[constants.PropertyStatus] = constants.StatusCreated

		// add the new cache to the config
		e.config.AppendResource(cache)

		// save the state
		resources.SaveState(e.config)
	}

	// finally we can process and create resources
	processErr := e.readAndProcessConfig(path, vars, variablesFile, e.createCallback)

	// save the state regardless of error
	stateErr := resources.SaveState(e.config)
	if stateErr != nil {
		e.log.Info("Unable to save state", "error", stateErr)
	}

	return e.config.Resources, processErr
}

// checks if a string exists in an array if not it appends and returns a new
// copy
func appendIfNotContains(existing []string, s string) []string {
	for _, v := range existing {
		if v == s {
			return existing
		}
	}

	return append(existing, s)
}

// Destroy the resources defined by the state
func (e *EngineImpl) Destroy() error {
	e.log.Info("Destroying resources")

	// load the state
	c, err := resources.LoadState()
	if err != nil {
		e.log.Debug("State file does not exist")
	}

	e.config = c

	// run through the graph and call the destroy callback
	// disabled resources are not included in this callback
	// image cache which is manually added by Apply process
	// should have the correct dependency graph to be
	// destroyed last
	err = e.config.Process(e.destroyCallback, true)
	if err != nil {

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

	var parseError error
	var parsedConfig *hclconfig.Config

	if path == "" {
		return nil
	}

	variablesFiles := []string{}
	if variablesFile != "" {
		variablesFiles = append(variablesFiles, variablesFile)
	}

	hclParser := resources.SetupHCLConfig(callback, variables, variablesFiles)

	if utils.IsHCLFile(path) {
		// ParseFile processes the HCL, builds a graph of resources then calls
		// the callback for each resource in order
		//
		// We are not using the returned config as the resources are added to the
		// state on the callback
		//
		// If the callback returns an error we need to save the state and exit
		parsedConfig, parseError = hclParser.ParseFile(path)
	} else {
		// ParseFolder processes the HCL, builds a graph of resources then calls
		// the callback for each resource in order
		//
		// We are not using the returned config as the resources are added to the
		// state on the callback
		//
		// If the callback returns an error we need to save the state and exit
		parsedConfig, parseError = hclParser.ParseDirectory(path)
	}

	// process is not called for disabled resources, add manually
	err := e.appendDisabledResources(parsedConfig)
	if err != nil {
		return parseError
	}

	// process is not called for module resources, add manually
	err = e.appendModuleResources(parsedConfig)
	if err != nil {
		return parseError
	}

	// destroy an resouces that might have been set to disabled
	err = e.destroyDisabledResources()
	if err != nil {
		return err
	}

	return parseError
}

// destroyDisabledResources destroys any resrouces that were created but
// have subsequently been set to disabled
func (e *EngineImpl) destroyDisabledResources() error {
	// we need to check if we have any disabbled resroucea that are marked
	// as created, this could be because the disabled state has changed
	// these respurces should be destroyed

	for _, r := range e.config.Resources {
		if r.Metadata().Disabled &&
			r.Metadata().Properties[constants.PropertyStatus] == constants.StatusCreated {

			p := e.getProvider(r, e.clients)
			if p == nil {
				r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
				return fmt.Errorf("unable to create provider for resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
			}

			// call destroy
			err := p.Destroy()
			if err != nil {
				r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
				return fmt.Errorf("unable to destroy resource Name: %s, Type: %s", r.Metadata().Name, r.Metadata().Type)
			}

			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusDisabled
		}
	}

	return nil
}

// appends disabled resources in the given config to the engines config
func (e *EngineImpl) appendDisabledResources(c *hclconfig.Config) error {
	if c == nil {
		return nil
	}

	for _, r := range c.Resources {
		if r.Metadata().Disabled {
			// if the resource already exists just set the status to disabled
			er, err := e.config.FindResource(types.FQDNFromResource(r).String())
			if err == nil {
				er.Metadata().Disabled = true
				continue
			}

			// otherwise if not found the resource to the state
			err = e.config.AppendResource(r)
			if err != nil {
				return fmt.Errorf("unable to add disabled resource: %s", err)
			}
		}
	}

	return nil
}

// appends module in the given config to the engines config
func (e *EngineImpl) appendModuleResources(c *hclconfig.Config) error {
	if c == nil {
		return nil
	}

	for _, r := range c.Resources {
		if r.Metadata().Type == types.TypeModule {
			// if the resource already exists remove it
			er, err := e.config.FindResource(types.FQDNFromResource(r).String())
			if err == nil {
				e.config.RemoveResource(er)
			}

			// add the resource to the state
			err = e.config.AppendResource(r)
			if err != nil {
				return fmt.Errorf("unable to add module resource: %s", err)
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

	// we need to check if a resource exists in the state, if so the status
	// should take precedence as all new resources will have an empty state
	sr, err := e.config.FindResource(r.Metadata().ID)
	if err == nil {
		// set the current status to the state status
		r.Metadata().Properties[constants.PropertyStatus] = sr.Metadata().Properties[constants.PropertyStatus]

		// remove the resource, we will add the new version to the state
		err = e.config.RemoveResource(r)
		if err != nil {
			return fmt.Errorf(`unable to remove resource "%s" from state, %s`, r.Metadata().ID, err)
		}
	}

	var providerError error
	switch r.Metadata().Properties[constants.PropertyStatus] {
	case constants.StatusCreated:
		providerError = p.Refresh()
		if providerError != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		}

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
		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusCreated
		providerError = p.Create()
		if providerError != nil {
			r.Metadata().Properties[constants.PropertyStatus] = constants.StatusFailed
		}
	}

	// add the resource to the state
	err = e.config.AppendResource(r)
	if err != nil {
		return fmt.Errorf(`unable add resource "%s" to state, %s`, r.Metadata().ID, err)
	}

	// did we just create a network, if so we need to attach the image cache
	// to the network and set the dependency
	if r.Metadata().Type == resources.TypeNetwork && r.Metadata().Properties[constants.PropertyStatus] == constants.StatusCreated {
		// get the image cache
		ic, err := e.config.FindResource("resource.image_cache.default")
		if err == nil {
			e.log.Debug("Attaching image cache to network", "network", ic.Metadata().ID)
			ic.Metadata().DependsOn = appendIfNotContains(ic.Metadata().DependsOn, r.Metadata().ID)

			// reload the networks
			np := e.getProvider(ic, e.clients)
			np.Create()
		} else {
			e.log.Error("Unable to find Image Cache", "error", err)
		}
	}

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
		return fmt.Errorf("unable to destroy resource Name: %s, Type: %s, Error: %s", r.Metadata().Name, r.Metadata().Type, err)
	}

	// remove from the state only if not errored
	e.config.RemoveResource(r)

	return nil
}
