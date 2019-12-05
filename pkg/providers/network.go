package providers

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/shipyard-run/cli/pkg/clients"
	"github.com/shipyard-run/cli/pkg/config"
)

// Network is a provider for creating docker networks
type Network struct {
	config *config.Network
	client clients.Docker
}

// NewNetwork creates a new network with the given config and Docker client
func NewNetwork(co *config.Network, cl clients.Docker) *Network {
	return &Network{co, cl}
}

// Create implements the provider interface method for creating new networks
func (n *Network) Create() error {
	opts := types.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				network.IPAMConfig{
					Subnet: n.config.Subnet,
				},
			},
		},
		Attachable: true,
	}

	_, err := n.client.NetworkCreate(context.Background(), n.config.Name, opts)
	return err
}

// Destroy implements the provider interface method for destroying networks
func (n *Network) Destroy() error {
	return n.client.NetworkRemove(context.Background(), n.config.Name)
}

// Lookup the ID for a network
func (n *Network) Lookup() (string, error) {
	return "", nil
}
