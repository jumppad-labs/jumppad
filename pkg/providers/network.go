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

	// validate the subnet
	_, cidr, err := net.ParseCIDR(n.config.Subnet)
	if err != nil {
		return fmt.Errorf("Unable to create network %s, invalid subnet %s", n.config.Name, n.config.Subnet)
	}

	// get all the networks
	nets, err := n.getNetworks("")
	if err != nil {
		return err
	}

	// is the network name and subnet equal to one which already exists
	for _, ne := range nets {
		if ne.Name == n.config.Name {
			for _, ci := range ne.IPAM.Config {
				// check that the returned networks subnet matches the existing networks subnet
				if ci.Subnet != n.config.Subnet {
					n.log.Info("Network already exists, skip creation", "ref", n.config.Name)
					return nil
				}
			}
		}
	}

	// check for overlapping subnets
	for _, ne := range nets {
		for _, ci := range ne.IPAM.Config {
			_, cidr2, err := net.ParseCIDR(ci.Subnet)
			if err != nil {
				// unable to parse the CIDR should not happen
				return err
			}

			if cidr.Contains(cidr2.IP) || cidr2.Contains(cidr.IP) {
				return fmt.Errorf("Unable to create network %s, Network %s already exists with an overlapping subnet %s. Either remove the network '%s' or change the subnet for your network", n.config.Name, ne.Name, ci.Subnet, ne.Name)
			}
		}
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
	nets, err := n.getNetworks(n.config.Name)

	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, n1 := range nets {
		ids = append(ids, n1.ID)
	}

	return ids, nil
}

func (n *Network) getNetworks(name string) ([]types.NetworkResource, error) {
	args := filters.NewArgs()
	args.Add("name", name)
	return n.client.NetworkList(context.Background(), types.NetworkListOptions{Filters: args})
}
