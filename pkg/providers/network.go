package providers

import (
	"context"
	"fmt"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
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

	// check if the network exists
	ids, err := n.Lookup()
	if err != nil {
		return err
	}

	// exists do not create
	if len(ids) > 0 {
		n.log.Info("Network already exists, skip creation", "ref", n.config.Name)
		return nil
	}

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

	_, err = n.client.NetworkCreate(context.Background(), n.config.Name, opts)
	if err != nil {
		return err
	}

	// set the state
	n.config.Status = config.Applied

	return err
}

// Destroy implements the provider interface method for destroying networks
func (n *Network) Destroy() error {
	n.log.Info("Destroy Network", "ref", n.config.Name)

	// check network exists if so remove
	ids, err := n.Lookup()
	if err != nil {
		return xerrors.Errorf("Unable to list networks: %w", err)
	}

	if len(ids) == 1 {
		return n.client.NetworkRemove(context.Background(), n.config.Name)
	}

	return nil
}

// Lookup the ID for a network
func (n *Network) Lookup() ([]string, error) {
	args := filters.NewArgs()
	nets, err := n.client.NetworkList(context.Background(), types.NetworkListOptions{Filters: args})
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, n1 := range nets {
		// is the network name equal to the config name
		if n1.ID == n.config.Name {
			// check that the returned networks subnet matches the existing networks subnet
			if n1.IPAM.Config[0].Subnet != n.config.Subnet {
				return nil, fmt.Errorf("Network %s already exists but with different subnet", n.config.Name)
			}

			ids = append(ids, n1.ID)
		} else {
			// if this is another network does the subnet overlap with the requested subnet if so return an error
			_, cidr1, err := net.ParseCIDR(n.config.Subnet)
			if err != nil {
				// unable to parse the CIDR should not happen
				return nil, err
			}

			for _, ci := range n1.IPAM.Config {
				_, cidr2, err := net.ParseCIDR(ci.Subnet)
				if err != nil {
					// unable to parse the CIDR should not happen
					return nil, err
				}

				if cidr1.Contains(cidr2.IP) || cidr2.Contains(cidr1.IP) {
					return nil, fmt.Errorf("Unable to create network %s, Network %s already exists with an overlapping subnet %s", n.config.Name, n1.ID, ci.Subnet)
				}
			}
		}
	}

	return ids, nil
}

// Config returns the config for the provider
func (c *Network) Config() ConfigWrapper {
	return ConfigWrapper{"config.Network", c.config}
}
