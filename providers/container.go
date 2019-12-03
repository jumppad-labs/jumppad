package providers

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/shipyard-run/cli/clients"
	"github.com/shipyard-run/cli/config"
)

// Container is a provider for creating and destroying Docker containers
type Container struct {
	config *config.Container
	client clients.Docker
}

// NewContainer creates a new container with the given config and Docker client
func NewContainer(co *config.Container, cl clients.Docker) *Container {
	return &Container{co, cl}
}

// Create implements provider method and creates a Docker container with the given config
func (c *Container) Create() error {

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
		Image:        c.config.Image,
		Env:          env,
		Cmd:          c.config.Command,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	// create the host and network configs
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	// Create volume mounts
	mounts := make([]mount.Mount, 0)
	for _, vc := range c.config.Volumes {

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: vc.Source,
			Target: vc.Destination,
		})
	}

	hc.Mounts = mounts

	// create the ports config
	ports := createPublishedPorts(c.config.Ports)
	dc.ExposedPorts = ports.ExposedPorts
	hc.PortBindings = ports.PortBindings

	// make sure the image name is canonical
	image := c.config.Image
	imageParts := strings.Split(image, "/")
	switch len(imageParts) {
	case 1:
		image = fmt.Sprintf("docker.io/library/%s", imageParts[0])
	case 2:
		image = fmt.Sprintf("docker.io/%s/%s", imageParts[0], imageParts[1])
	}

	out, err := c.client.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, out)

	cont, err := c.client.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		FQDN(c.config.Name, c.config.NetworkRef.Name),
	)
	if err != nil {
		return err
	}

	return c.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
}

// Destroy stops and removes the container
func (c *Container) Destroy() error {
	id, err := c.Lookup()
	if err != nil {
		return err
	}

	// container not running do nothing
	if id == "" {
		return nil
	}

	return c.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
}

// Lookup the containers ID based on the config
func (c *Container) Lookup() (string, error) {
	name := FQDN(c.config.Name, c.config.NetworkRef.Name)

	args, _ := filters.ParseFlag("name="+name, filters.NewArgs())

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
