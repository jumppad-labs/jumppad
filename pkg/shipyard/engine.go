package shipyard

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/mitchellh/mapstructure"
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
	providers         [][]providers.Provider
	clients           *Clients
	config            *config.Config
	log               hclog.Logger
	generateProviders generateProvidersFunc
	stateLock         sync.Mutex
	state             []providers.ConfigWrapper
}

type generateProvidersFunc func(c *config.Config, cl *Clients, l hclog.Logger) [][]providers.Provider

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

	// if we are destroying all resources set the pending modification flag
	if allResources {
		for _, gp := range e.providers {
			for _, p := range gp {
				p.SetState(config.PendingModification)
			}
		}
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
	cc, err := config.New()
	if err != nil {
		return err
	}

	if path != "" {
		if utils.IsHCLFile(path) {
			err = config.ParseHCLFile(path, cc)
			if err != nil {
				return err
			}
		} else {
			err = config.ParseFolder(path, cc)
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

	e.providers = e.generateProviders(e.config, e.clients, e.log)

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
	errs := make(chan error)
	done := make(chan struct{})

	// create the wait group and set the size to the provider length
	wg := sync.WaitGroup{}
	wg.Add(len(p))

	for _, pr := range p {
		go func(pr providers.Provider) {
			defer wg.Done()

			// only attempt to create if the state is awaiting creation
			if pr.State() == config.PendingCreation {
				err := pr.Create()
				if err != nil {
					errs <- err
					return
				}
			}

			// if an error happens then the state will end up incomplete
			pr.SetState(config.Applied)

			// append the state
			e.stateLock.Lock()
			defer e.stateLock.Unlock()
			e.state = append(e.state, pr.Config())
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
func (e *Engine) destroyParallel(p []providers.Provider) error {
	// create the wait group and set the size to the provider length
	wg := sync.WaitGroup{}
	wg.Add(len(p))

	for _, pr := range p {
		go func(pr providers.Provider) {
			defer wg.Done()

			if pr.State() == config.PendingModification {
				pr.Destroy()
				return
			}

			// only add to the state if we did not delete
			e.stateLock.Lock()
			defer e.stateLock.Unlock()
			e.state = append(e.state, pr.Config())
		}(pr)
	}

	wg.Wait()

	return nil
}

// save state serializes the state file into json formatted file
func (e *Engine) saveState() error {
	e.log.Info("Writing state file")

	sd := utils.StateDir()
	sp := utils.StatePath()

	// if it does not exist create the state folder
	_, err := os.Stat(sd)
	if err != nil {
		os.MkdirAll(sd, os.ModePerm)
	}

	// if the statefile exists overwrite it
	_, err = os.Stat(sp)
	if err == nil {
		// delete the old state
		os.Remove(sp)
	}

	// serialize the state to json and write to a file
	f, err := os.Create(sp)
	if err != nil {
		e.log.Error("Unable to create state", "error", err)
		return err
	}
	defer f.Close()

	ne := json.NewEncoder(f)
	return ne.Encode(e.state)
}

func (e *Engine) configFromState(path string) (*config.Config, error) {
	cc := &config.Config{}

	// it is fine that the state might not exist
	f, err := os.Open(path)
	if err != nil {
		e.log.Debug("State file does not exist", "err", err)
		return cc, nil
	}
	defer f.Close()

	s := []interface{}{}
	jd := json.NewDecoder(f)
	jd.Decode(&s)

	// for each item set the config
	for _, c := range s {
		switch c.(map[string]interface{})["Type"].(string) {
		case "config.Network":
			n := &config.Network{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			// do not add the wan as this is automatically created
			if n.Name == "wan" {
				cc.WAN = n
			} else {
				cc.Networks = append(cc.Networks, n)
			}
		case "config.Docs":
			n := &config.Docs{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.Docs = n
		case "config.Cluster":
			fmt.Println("cluster")
			n := &config.Cluster{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.Clusters = append(cc.Clusters, n)
		case "config.Container":
			n := &config.Container{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.Containers = append(cc.Containers, n)
		case "config.Helm":
			n := &config.Helm{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.HelmCharts = append(cc.HelmCharts, n)
		case "config.K8sConfig":
			n := &config.K8sConfig{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.K8sConfig = append(cc.K8sConfig, n)
		case "config.Ingress":
			n := &config.Ingress{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.Ingresses = append(cc.Ingresses, n)
		case "config.LocalExec":
			n := &config.LocalExec{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.LocalExecs = append(cc.LocalExecs, n)
		case "config.RemoteExec":
			n := &config.RemoteExec{}
			err := mapstructure.Decode(c.(map[string]interface{})["Value"].(interface{}), &n)
			if err != nil {
				return nil, err
			}

			cc.RemoteExecs = append(cc.RemoteExecs, n)
		}
	}

	return cc, nil
}

// merge config items merges the two configs together removing duplicates
func (e *Engine) mergeConfigItems(c *config.Config, state *config.Config, delete bool) *config.Config {
	ns := *state

	// process the clusters
	for _, sc := range c.Clusters {
		found := -1
		for n, i := range state.Clusters {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			// dont add to the collection if the item is not found and we are deleting
			// else the the item will end up in the state
			if !delete {
				ns.Clusters = append(ns.Clusters, sc)
			}
		} else {
			e.log.Debug("Cluster already exists in state, update status", "name", sc.Name)
			ns.Clusters[found].State = config.PendingModification
		}
	}

	// process the containers
	for _, sc := range c.Containers {
		found := -1
		for n, i := range state.Containers {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.Containers = append(ns.Containers, sc)
			}
		} else {
			e.log.Debug("Container already exists in state, update status", "name", sc.Name)
			ns.Containers[found].State = config.PendingModification
		}
	}

	// process the networks
	for _, sc := range c.Networks {
		found := -1
		for n, i := range state.Networks {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.Networks = append(ns.Networks, sc)
			}
		} else {
			e.log.Debug("Network already exists in state, update status", "name", sc.Name)
			ns.Networks[found].State = config.PendingModification
		}
	}

	// process the helm charts
	for _, sc := range c.HelmCharts {
		found := -1
		for n, i := range state.HelmCharts {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.HelmCharts = append(ns.HelmCharts, sc)
			}
		} else {
			e.log.Debug("Helm chart already exists in state, update status", "name", sc.Name)
			ns.HelmCharts[found].State = config.PendingModification
		}
	}

	// process the kube config
	for _, sc := range c.K8sConfig {
		found := -1
		for n, i := range state.K8sConfig {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.K8sConfig = append(ns.K8sConfig, sc)
			}
		} else {
			e.log.Debug("Kubernetes config already exists in state, update status", "name", sc.Name)
			ns.K8sConfig[found].State = config.PendingModification
		}
	}

	// process the ingresses
	for _, sc := range c.Ingresses {
		found := -1
		for n, i := range state.Ingresses {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.Ingresses = append(ns.Ingresses, sc)
			}
		} else {
			e.log.Debug("Ingress already exists in state, update status", "name", sc.Name)
			ns.Ingresses[found].State = config.PendingModification
		}
	}

	// process the local
	for _, sc := range c.LocalExecs {
		found := -1
		for n, i := range state.LocalExecs {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.LocalExecs = append(ns.LocalExecs, sc)
			}
		} else {
			e.log.Debug("Ingress already exists in state, update status", "name", sc.Name)
			ns.LocalExecs[found].State = config.PendingModification
		}
	}

	// process the local
	for _, sc := range c.RemoteExecs {
		found := -1
		for n, i := range state.RemoteExecs {
			if i.Name == sc.Name {
				found = n
				break
			}
		}

		if found == -1 {
			if !delete {
				ns.RemoteExecs = append(ns.RemoteExecs, sc)
			}
		} else {
			e.log.Debug("Ingress already exists in state, update status", "name", sc.Name)
			ns.RemoteExecs[found].State = config.PendingModification
		}
	}

	if state.WAN != nil {
		e.log.Debug("WAN Network already exists in state, ignoring from apply", "name", state.WAN.Name)
	} else {
		if !delete {
			ns.WAN = c.WAN
		}
	}

	return &ns
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

	if c.WAN != nil {
		p := providers.NewNetwork(c.WAN, cc.Docker, l)
		oc[0] = append(oc[0], p)
	}

	for _, n := range c.Networks {
		p := providers.NewNetwork(n, cc.Docker, l)
		oc[0] = append(oc[0], p)
	}

	for _, c := range c.Containers {
		p := providers.NewContainer(*c, cc.ContainerTasks, l)
		oc[1] = append(oc[1], p)
	}

	for _, c := range c.Ingresses {
		p := providers.NewIngress(*c, cc.ContainerTasks, l)
		oc[1] = append(oc[1], p)
	}

	if c.Docs != nil {
		p := providers.NewDocs(c.Docs, cc.ContainerTasks, l)
		oc[1] = append(oc[1], p)
	}

	for _, c := range c.Clusters {
		p := providers.NewCluster(*c, cc.ContainerTasks, cc.Kubernetes, cc.HTTP, l)
		oc[2] = append(oc[2], p)
	}

	for _, c := range c.HelmCharts {
		p := providers.NewHelm(c, cc.Kubernetes, cc.Helm, l)
		oc[3] = append(oc[3], p)
	}

	for _, c := range c.K8sConfig {
		p := providers.NewK8sConfig(c, cc.Kubernetes, l)
		oc[4] = append(oc[4], p)
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
