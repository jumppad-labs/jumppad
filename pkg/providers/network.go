package providers

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Network is a provider for creating docker networks
type Network struct {
	config *config.Network
	client clients.Docker
	log    hclog.Logger
}

// NewNetwork creates a new network with the given config and Docker client
func NewNetwork(co *config.Network, cl clients.Docker, l hclog.Logger) *Network {
	return &Network{co, cl, l}
}

// Create implements the provider interface method for creating new networks
func (n *Network) Create() error {
	n.log.Info("Creating Network", "ref", n.config.Name)

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
	n.log.Info("Destroying Network", "ref", n.config.Name)

	return n.client.NetworkRemove(context.Background(), n.config.Name)
}

// Lookup the ID for a network
func (n *Network) Lookup() (string, error) {
	return "", nil
}
