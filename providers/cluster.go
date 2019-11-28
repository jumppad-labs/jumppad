package providers

import (
	"github.com/shipyard-run/cli/config"
	"github.com/shipyard-run/cli/clients"
)

// Cluster defines a provider which can create a cluster
type Cluster struct {
	Config    *config.Cluster
	ClientK3s clients.K3s
}

func (c *Cluster) Create() {

}
