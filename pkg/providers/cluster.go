package providers

import (
	"errors"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

var (
	ErrorClusterDriverNotImplemented = errors.New("driver not implemented")
	ErrorClusterExists               = errors.New("cluster exists")
)

// Cluster defines a provider which can create a cluster
type Cluster struct {
	config     config.Cluster
	client     clients.ContainerTasks
	kubeClient clients.Kubernetes
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewCluster creates a new
func NewCluster(c config.Cluster, cc clients.ContainerTasks, kc clients.Kubernetes, hc clients.HTTP, l hclog.Logger) *Cluster {
	return &Cluster{c, cc, kc, hc, l}
}

// Create implements interface method to create a cluster of the specified type
func (c *Cluster) Create() error {
	switch c.config.Driver {
	case "k3s":
		return c.createK3s()
	case "nomad":
		return c.createNomad()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Destroy implements interface method to destroy a cluster
func (c *Cluster) Destroy() error {
	switch c.config.Driver {
	case "k3s":
		return c.destroyK3s()
	case "nomad":
		return c.destroyNomad()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Lookup the a clusters current state
func (c *Cluster) Lookup() ([]string, error) {
	/*
		// lookup the server id
		// base of cluster is a container
		co := &config.Container{
			Name:       c.config.Name,
			NetworkRef: c.config.NetworkRef,
		}

		p := NewContainer(co, c.client, c.log.With("parent_ref", c.config.Name))

		return p.Lookup()
	*/
	return []string{}, nil
}

// Config returns the config for the provider
func (c *Cluster) Config() ConfigWrapper {
	return ConfigWrapper{"config.Cluster", c.config}
}
