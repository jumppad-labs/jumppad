package providers

import (
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// DOKSCluster defines the interaction for creating, and destroying
// Kubernetes clusters in Digital Ocean cloud.
type DOKSCluster struct {
	config     *config.K8sCluster
	client     clients.ContainerTasks
	kubeClient clients.Kubernetes
	httpClient clients.HTTP
	log        hclog.Logger
}

// Create a new DOKS cluster in Digial Ocean
func (d *DOKSCluster) Create() error {
	return nil
}

// Destroy the cluster
func (d *DOKSCluster) Destroy() error {
	return nil
}

// Lookup the clusters ID
func (d *DOKSCluster) Lookup() ([]string, error) {
	return nil, nil
}
