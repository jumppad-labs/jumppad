package providers

import (
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
)

// Cluster defines a provider which can create a cluster
type Cluster struct {
	Config    *config.Cluster
	ClientK3s clients.K3s
}

// Create implements interface method to create a cluster
func (c *Cluster) Create() {

}
