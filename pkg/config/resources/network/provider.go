package network

import (
	"context"
	"fmt"
	"net"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &Provider{}

// Network is a provider for creating docker networks
type Provider struct {
	config *Network
	client container.Docker
	log    sdk.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Network)
	if !ok {
		return fmt.Errorf("unable to initialize Network provider, resource is not of type Network")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.Docker
	p.log = l

	return nil
}

// Create implements the provider interface method for creating new networks
func (p *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Network", "ref", p.config.Meta.ID)

	// validate the subnet
	_, cidr, err := net.ParseCIDR(p.config.Subnet)
	if err != nil {
		return fmt.Errorf("unable to create network %s, invalid subnet %s", p.config.Meta.Name, p.config.Subnet)
	}

	// check the local networks for overlapping subnets
	hostIPs, err := p.getHostIPs()
	if err != nil {
		return fmt.Errorf("unable to query host networks: %s", err)
	}

	for _, n := range hostIPs {
		if cidr.Contains(n) {
			return fmt.Errorf("unable to create network %s, a local ip address %s already exists that overlaps with the subnet %s. Please use a network subnet that does not confict with a local range", p.config.Meta.Name, n, p.config.Subnet)
		}
	}

	// get all the networks
	nets, err := p.getNetworks("")
	if err != nil {
		return fmt.Errorf("unable to list existing networks: %s. If you are using podman, ensure that the default 'podman' network exists", err)
	}

	// is the network name and subnet equal to one which already exists
	for _, ne := range nets {
		if ne.Name == p.config.Meta.Name {
			return fmt.Errorf("a Network already exists with the name: %s ref:%s", p.config.Meta.Name, p.config.Meta.ID)
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
				return fmt.Errorf("unable to create network %s, Network %s already exists with an overlapping subnet %s. Either remove the network '%s' or change the subnet for your network", p.config.Meta.Name, ne.Name, ci.Subnet, ne.Name)
			}
		}
	}

	// check the network drivers, if bridge is available use bridge, else use nat
	p.log.Debug("Attempting to create using bridge plugin", "ref", p.config.Meta.Name)
	err = p.createWithDriver("bridge")
	if err != nil {
		p.log.Debug("Unable to create using bridge, fall back to use nat plugin", "ref", p.config.Meta.Name, "error", err)
		// fall back to nat
		err = p.createWithDriver("nat")
		if err != nil {
			return err
		}
	}

	return err
}

// Destroy implements the provider interface method for destroying networks
func (p *Provider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Network", "ref", p.config.Meta.ID)

	// check network exists if so remove
	ids, err := p.Lookup()
	if err != nil {
		return fmt.Errorf("unable to list networks: %w", err)
	}

	if len(ids) == 1 {
		networkID := ids[0]
		// check if the network has containers attached
		containers, err := p.getConnectedContainers(networkID)
		if err != nil {
			return fmt.Errorf("unable to list connected containers: %w", err)
		}

		for _, containerID := range containers {
			err := p.client.NetworkDisconnect(context.Background(), networkID, containerID, true)
			if err != nil {
				return fmt.Errorf("unable to disconnect container from network: %w", err)
			}
		}

		return p.client.NetworkRemove(context.Background(), p.config.Meta.Name)
	}

	return nil
}

func (p *Provider) getConnectedContainers(id string) ([]string, error) {
	containers := []string{}

	summary, err := p.client.NetworkInspect(context.Background(), id, network.InspectOptions{})
	if err != nil {
		return containers, err
	}

	for containerID := range summary.Containers {
		containers = append(containers, containerID)
	}

	return containers, nil
}

// Lookup the ID for a network
func (p *Provider) Lookup() ([]string, error) {
	nets, err := p.getNetworks(p.config.Meta.Name)

	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, n1 := range nets {
		ids = append(ids, n1.ID)
	}

	return ids, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Network", "ref", p.config.Meta.ID)

	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}

func (p *Provider) createWithDriver(driver string) error {
	opts := network.CreateOptions{
		// CheckDuplicate: true,
		Driver:     driver,
		EnableIPv6: &p.config.EnableIPv6,
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet: p.config.Subnet,
				},
			},
		},
		Labels: map[string]string{
			"created_by": "jumppad",
			"id":         p.config.Meta.ID,
		},
		Attachable: true,
	}

	_, err := p.client.NetworkCreate(context.Background(), p.config.Meta.Name, opts)

	return err
}

func (p *Provider) getNetworks(name string) ([]network.Summary, error) {
	args := filters.NewArgs()
	args.Add("name", name)
	return p.client.NetworkList(context.Background(), network.ListOptions{Filters: args})
}

func (p *Provider) getHostIPs() ([]net.IP, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ips := []net.IP{}

	for _, i := range ifs {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if ip := v.IP.To4(); ip != nil {
					ips = append(ips, ip)
				}
			case *net.IPAddr:
				if ip := v.IP.To4(); ip != nil {
					ips = append(ips, ip)
				}
			}
		}
	}

	return ips, nil
}
