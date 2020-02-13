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

// K8sCluster defines a provider which can create Kubernetes clusters
type K8sCluster struct {
	config     config.K8sCluster
	client     clients.ContainerTasks
	kubeClient clients.Kubernetes
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewK8sCluster creates a new Kubernetes cluster provider
func NewK8sCluster(c config.K8sCluster, cc clients.ContainerTasks, kc clients.Kubernetes, hc clients.HTTP, l hclog.Logger) *K8sCluster {
	return &K8sCluster{c, cc, kc, hc, l}
}

// Create implements interface method to create a cluster of the specified type
func (c *K8sCluster) Create() error {
	switch c.config.Driver {
	case "k3s":
		return c.createK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Destroy implements interface method to destroy a cluster
func (c *K8sCluster) Destroy() error {
	switch c.config.Driver {
	case "k3s":
		return c.destroyK3s()
	default:
		return ErrorClusterDriverNotImplemented
	}
}

// Lookup the a clusters current state
func (c *K8sCluster) Lookup() ([]string, error) {
	return []string{}, nil
}
