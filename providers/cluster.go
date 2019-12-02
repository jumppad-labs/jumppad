package providers

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
)

var (
	ErrorClusterDriverNotImplemented = errors.New("driver not implemented")
	ErrorClusterExists               = errors.New("cluster exists")
)

// Cluster defines a provider which can create a cluster
type Cluster struct {
	config *config.Cluster
	client clients.Docker
}

// NewCluster creates a new
func NewCluster(c *config.Cluster, cc clients.Docker) *Cluster {
	return &Cluster{c, cc}
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
	args := filters.NewArgs()
	args.Add("label", "app=shipyard")
	args.Add("label", "component=server")
	args.Add("label", "cluster="+c.config.Name)

	opts := types.ContainerListOptions{Filters: args}

	cl, err := c.client.ContainerList(context.Background(), opts)
	if err != nil {
		return "", err
	}

	return cl[0].ID, nil
}
