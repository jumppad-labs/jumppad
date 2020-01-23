package providers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config *config.Container
	client clients.Docker
	log    hclog.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co *config.Container, cl clients.Docker, l hclog.Logger) *Container {
	return &Container{co, cl, l}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {
	c.log.Info("Creating Container", "ref", c.config.Name)

	// create a unique name based on service network [container].[network].shipyard
	// attach to networks
	// - networkRef
	// - wanRef

	// convert the environment vars to a list of [key]=[value]
	env := make([]string, 0)
	for _, kv := range c.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
	}

	// create the container config
	dc := &container.Config{
		Hostname:     c.config.Name,
		Image:        c.config.Image.Name,
		Env:          env,
		Cmd:          c.config.Command,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	// create the host and network configs
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	// attach the container to the network
	nc.EndpointsConfig = make(map[string]*network.EndpointSettings)
	if c.config.NetworkRef != nil {
		nc.EndpointsConfig[c.config.NetworkRef.Name] = &network.EndpointSettings{NetworkID: c.config.NetworkRef.Name}

		// are we binding to a specific ip
		if c.config.IPAddress != "" {
			c.log.Debug("Assigning static ip address", "ref", c.config.Name, "ip_address", c.config.IPAddress)
			//nc.EndpointsConfig[c.config.NetworkRef.Name].IPAddress = c.config.IPAddress
			nc.EndpointsConfig[c.config.NetworkRef.Name].IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: c.config.IPAddress}
		}
	}

	// Create volume mounts
	mounts := make([]mount.Mount, 0)
	for _, vc := range c.config.Volumes {

		// default mount type to bind
		t := mount.TypeBind

		// select mount type if set
		switch vc.Type {
		case "bind":
			t = mount.TypeBind
		case "volume":
			t = mount.TypeVolume
		case "tmpfs":
			t = mount.TypeTmpfs
		}

		mounts = append(mounts, mount.Mount{
			Type:   t,
			Source: vc.Source,
			Target: vc.Destination,
		})
	}

	hc.Mounts = mounts

	// create the ports config
	ports := createPublishedPorts(c.config.Ports)
	dc.ExposedPorts = ports.ExposedPorts
	hc.PortBindings = ports.PortBindings

	// is this a privlidged container
	hc.Privileged = c.config.Privileged

	// make sure the image name is canonical
	err := pullImage(c.client, c.config.Image, c.log.With("parent_ref", c.config.Name))
	if err != nil {
		return err
	}

	cont, err := c.client.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		FQDN(c.config.Name, c.config.NetworkRef),
	)
	if err != nil {
		return err
	}

	return c.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	c.log.Info("Destroy Container", "ref", c.config.Name)

	id, err := c.Lookup()
	if err != nil {
		return err
	}

	// container not running do nothing
	if id == "" {
		return nil
	}

	return c.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
}

// Lookup the containers ID based on the config
func (c *Container) Lookup() (string, error) {
	name := c.config.Name
	if c.config.NetworkRef != nil {
		name = FQDN(c.config.Name, c.config.NetworkRef)
	}

	args := filters.NewArgs()
	args.Add("name", name)

	opts := types.ContainerListOptions{Filters: args, All: true}

	cl, err := c.client.ContainerList(context.Background(), opts)
	if err != nil {
		return "", err
	}

	if len(cl) > 0 {
		return cl[0].ID, nil
	}

	return "", nil
}

// publishedPorts defines a Docker published port
type publishedPorts struct {
	ExposedPorts map[nat.Port]struct{}
	PortBindings map[nat.Port][]nat.PortBinding
}

// createPublishedPorts converts a list of config.Port to Docker publishedPorts type
func createPublishedPorts(ps []config.Port) publishedPorts {
	pp := publishedPorts{
		ExposedPorts: make(map[nat.Port]struct{}, 0),
		PortBindings: make(map[nat.Port][]nat.PortBinding, 0),
	}

	for _, p := range ps {
		dp, _ := nat.NewPort(p.Protocol, strconv.Itoa(p.Local))
		pp.ExposedPorts[dp] = struct{}{}

		pb := []nat.PortBinding{
			nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(p.Host),
			},
		}

		pp.PortBindings[dp] = pb
	}

	return pp
}
