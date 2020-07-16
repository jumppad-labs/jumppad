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
	DependsOn []string `json:"depends_on,omitempty"`
	// Module is the name of the module if a resource has been loaded from a module
	Module string `json:"module,omitempty"`

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

// FindResources returns an array of resources for the given module
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
// name is defined with the convention [type].[name]
// if a resource can not be found resource will be null and an
// error will be returned
//
// e.g. to find a cluster named k3s
// r, err := c.FindResource("cluster.k3s")
func (c *Config) FindResource(name string) (Resource, error) {
	parts := strings.Split(name, ".")
	for _, r := range c.Resources {
		if r.Info().Type == ResourceType(parts[0]) && r.Info().Name == parts[1] {
			return r, nil
		}
	}

	return nil, ResourceNotFoundError{name}
}

// AddResource adds a given resource to the resource list
// if the resource already exists an error will be returned
func (c *Config) AddResource(r Resource) error {
	rf, err := c.FindResource(fmt.Sprintf("%s.%s", r.Info().Type, r.Info().Name))
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
