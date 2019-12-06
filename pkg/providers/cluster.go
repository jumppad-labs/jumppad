package providers

import (
	"errors"

	"github.com/shipyard-run/cli/pkg/clients"
	"github.com/shipyard-run/cli/pkg/config"
)

var (
	ErrorClusterDriverNotImplemented = errors.New("driver not implemented")
	ErrorClusterExists               = errors.New("cluster exists")
)

// Cluster defines a provider which can create a cluster
type Cluster struct {
	config     *config.Cluster
	client     clients.Docker
	kubeClient clients.Kubernetes
}

// NewCluster creates a new
func NewCluster(c *config.Cluster, cc clients.Docker, kc clients.Kubernetes) *Cluster {
	return &Cluster{c, cc, kc}
}

// Create implements interface method to create a cluster of the specified type
func (c *Cluster) Create() error {
	switch c.config.Driver {
	case "k3s":
		return c.createK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Destroy implements interface method to destroy a cluster
func (c *Cluster) Destroy() error {
	switch c.config.Driver {
	case "k3s":
		return c.destroyK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Lookup the a clusters current state
func (c *Cluster) Lookup() (string, error) {
	// lookup the server id
	// base of cluster is a container
	co := &config.Container{
		Name:       c.config.Name,
		NetworkRef: c.config.NetworkRef,
	}

	p := NewContainer(co, c.client)

	return p.Lookup()
}
