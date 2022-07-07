package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/dag"
)

// Status defines the current state of a resource
type Status string

// ResourceType is the type of the resource
type ResourceType string

// Applied means the resource has been successfully created
const Applied Status = "applied"

// PendingCreation means the resource has not yet been created
// it will be created on the next run
const PendingCreation Status = "pending_creation"

// PendingModification means the resource has been created but
// if the action is Apply then the resource will be re-created with the next run
// if the action is Delete then the resource will be removed with the next run
const PendingModification Status = "pending_modification"

// PendingUpdate means the resource has been requested to be updated
// if the action is Apply then the resource will be ignored with the next run
// if the action is Delete then the resource will be removed with the next run
const PendingUpdate Status = "pending_update"

// Failed means the resource failed during creation
// if the action is Apply the resource will be re-created at the next run
const Failed Status = "failed"

// Destroyed means the resource has been destroyed
const Destroyed Status = "destroyed"

// Disabled means the resource will be ignored by the engine and no resources
// will be created or destroyed
const Disabled Status = "disabled"

type Resource interface {
	Info() *ResourceInfo
	FindDependentResource(string) (Resource, error)
	AddChild(Resource)
}

// ResourceInfo is the embedded type for any config resources
type ResourceInfo struct {
	// Name is the name of the resource
	Name string `json:"name"`
	// Type is the type of resource, this is the text representation of the golang type
	Type ResourceType `json:"type"`
	// Status is the current status of the resource, this is always PendingCreation initially
	Status Status `json:"status,omitempty"`
	// DependsOn is a list of objects which must exist before this resource can be applied
	DependsOn []string `json:"depends_on,omitempty" mapstructure:"depends_on"`
	// Module is the name of the module if a resource has been loaded from a module
	Module string `json:"module,omitempty"`
	// Enabled determines if a resource is enabled and should be processed
	Disabled bool `hcl:"disabled,optional" json:"disabled,omitempty"`

	// parent container
	Config *Config `json:"-"`
}

func (r *ResourceInfo) Info() *ResourceInfo {
	return r
}

func (r *ResourceInfo) FindDependentResource(name string) (Resource, error) {
	return r.Config.FindResource(name)
}

func (r *ResourceInfo) AddChild(c Resource) {
	// copy the config reference so the child can lookup resources
	c.Info().Config = r.Config

	// override the childs type so that the names are created correctly
	c.Info().Type = r.Type
}

// Config defines the stack config
type Config struct {
	Blueprint *Blueprint `json:"blueprint"`
	Resources []Resource `json:"resources"`
}

// ResourceNotFoundError is thrown when a resource could not be found
type ResourceNotFoundError struct {
	Name string
}

func (e ResourceNotFoundError) Error() string {
	return fmt.Sprintf("Resource not found: %s", e.Name)
}

// ResourceExistsError is thrown when a resource already exists in the resource list
type ResourceExistsError struct {
	Name string
}

func (e ResourceExistsError) Error() string {
	return fmt.Sprintf("Resource already exists: %s", e.Name)
}

// New creates a new Config
func New() *Config {
	c := &Config{}

	return c
}

// FindModuleResources returns an array of resources for the given module
func (c *Config) FindModuleResources(name string) ([]Resource, error) {
	resources := []Resource{}

	parts := strings.Split(name, ".")

	for _, r := range c.Resources {
		if r.Info().Module == parts[1] {
			resources = append(resources, r)
		}
	}

	if len(resources) > 0 {
		return resources, nil
	}

	return nil, ResourceNotFoundError{name}
}

// FindResource returns the resource for the given name
// name is defined with the convention [module].[type].[name]
// if a resource can not be found resource will be null and an
// error will be returned
//
// e.g. to find a cluster named k3s
// r, err := c.FindResource("cluster.k3s")
//
// simple.consul.container.consul
func (c *Config) FindResource(name string) (Resource, error) {
	parts := strings.Split(name, ".")

	typeLoc := -1
	// find the type
	for i, p := range parts {
		if isRegisteredType(ResourceType(p)) {
			typeLoc = i
		}
	}

	if typeLoc == -1 {
		return nil, fmt.Errorf("unable to find resource %s, invalid type", name)
	}

	module := ""
	if typeLoc > 0 {
		module = strings.Join(parts[:typeLoc], ".")
	}

	// the name could contain . so join after the first
	typ := parts[typeLoc]
	n := strings.Join(parts[typeLoc+1:], ".")

	// this is an internal error and should not happen unless there is an issue with a provider
	// there was, hence why we are here
	if c.Resources == nil {
		return nil, fmt.Errorf("unable to find resources, reference to parent config does not exist. Ensure that the object has been added to the config: `config.ResourceInfo.AddChild(type)`")
	}

	for _, r := range c.Resources {
		if r.Info().Module == module &&
			r.Info().Type == ResourceType(typ) &&
			r.Info().Name == n {
			return r, nil
		}
	}

	return nil, ResourceNotFoundError{name}
}

// FindResourcesByType returns the resources from the given type
func (c *Config) FindResourcesByType(t string) []Resource {
	res := []Resource{}

	for _, r := range c.Resources {
		if r.Info().Type == ResourceType(t) {
			res = append(res, r)
		}
	}

	return res
}

// AddResource adds a given resource to the resource list
// if the resource already exists an error will be returned
func (c *Config) AddResource(r Resource) error {
	rn := fmt.Sprintf("%s.%s", r.Info().Name, r.Info().Type)
	if r.Info().Module != "" {
		rn = fmt.Sprintf("%s.%s.%s", r.Info().Module, r.Info().Type, r.Info().Name)
	}

	rf, err := c.FindResource(rn)
	if err == nil && rf != nil {
		return ResourceExistsError{r.Info().Name}
	}

	r.Info().Config = c
	c.Resources = append(c.Resources, r)

	return nil
}

func (c *Config) RemoveResource(rf Resource) error {
	pos := -1
	for i, r := range c.Resources {
		if rf == r {
			pos = i
			break
		}
	}

	// found the resource remove from the collection
	// preserve order
	if pos > -1 {
		c.Resources = append(c.Resources[:pos], c.Resources[pos+1:]...)
		return nil
	}

	return ResourceNotFoundError{}
}

// DoYaLikeDAGs? dags? yeah dags! oh dogs.
// https://www.youtube.com/watch?v=ZXILzUpVx7A&t=0s
func (c *Config) DoYaLikeDAGs() (*dag.AcyclicGraph, error) {
	// create root node
	root := &Blueprint{}

	graph := &dag.AcyclicGraph{}
	graph.Add(root)

	// Loop over all resources and add to dag
	for _, resource := range c.Resources {
		//fmt.Printf("Resource: %s, Type: %s\n", resource.Info().Name, resource.Info().Type)
		graph.Add(resource)
	}

	// Add dependencies for all resources
	for _, resource := range c.Resources {
		hasDeps := false
		for _, d := range resource.Info().DependsOn {
			var err error
			dependencies := []Resource{}

			if strings.HasPrefix(d, "module.") {
				// find dependencies from modules
				dependencies, err = c.FindModuleResources(d)
				if err != nil {
					return nil, err
				}
			} else {
				// find dependencies for direct resources
				r, err := c.FindResource(d)
				if err != nil {
					return nil, err
				}
				dependencies = append(dependencies, r)
			}

			for _, d := range dependencies {
				hasDeps = true
				graph.Connect(dag.BasicEdge(d, resource))
			}
		}

		// if no deps add to root node
		if !hasDeps {
			graph.Connect(dag.BasicEdge(root, resource))
		}
	}

	return graph, nil
}

// ResourceCount defines the number of resources in a config
func (c *Config) ResourceCount() int {
	return len(c.Resources)
}
