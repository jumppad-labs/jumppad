package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// DockerTasks is a concrete implementation of ContainerTasks which uses the Docker SDK
type DockerTasks struct {
	c Docker
	l hclog.Logger
}

// NewDockerTasks creates a DockerTasks with the given Docker client
func NewDockerTasks(c Docker, l hclog.Logger) *DockerTasks {
	return &DockerTasks{c, l}
}

// CreateContainer creates a new Docker container for the given configuation
func (d *DockerTasks) CreateContainer(c config.Container) (string, error) {
	d.l.Info("Creating Container", "ref", c.Name)

	// create a unique name based on service network [container].[network].shipyard
	// attach to networks
	// - networkRef
	// - wanRef

	// convert the environment vars to a list of [key]=[value]
	env := make([]string, 0)
	for _, kv := range c.Environment {
		env = append(env, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
	}

	// create the container config
	dc := &container.Config{
		Hostname:     c.Name,
		Image:        c.Image.Name,
		Env:          env,
		Cmd:          c.Command,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	// create the host and network configs
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	nc.EndpointsConfig = make(map[string]*network.EndpointSettings)

	// attach the container to the network
	networkName := ""
	if c.NetworkRef != nil {
		nc.EndpointsConfig[c.NetworkRef.Name] = &network.EndpointSettings{NetworkID: c.NetworkRef.Name}

		// are we binding to a specific ip
		if c.IPAddress != "" {
			d.l.Debug("Assigning static ip address", "ref", c.Name, "ip_address", c.IPAddress)
			//nc.EndpointsConfig[c.config.NetworkRef.Name].IPAddress = c.config.IPAddress
			nc.EndpointsConfig[c.NetworkRef.Name].IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: c.IPAddress}
		}

		networkName = c.NetworkRef.Name
	}

	// attach the container to the WAN network
	if c.WANRef != nil {
		nc.EndpointsConfig[c.WANRef.Name] = &network.EndpointSettings{NetworkID: c.WANRef.Name}
	}

	// Create volume mounts
	mounts := make([]mount.Mount, 0)
	for _, vc := range c.Volumes {

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
	ports := createPublishedPorts(c.Ports)
	dc.ExposedPorts = ports.ExposedPorts
	hc.PortBindings = ports.PortBindings

	// is this a privlidged container
	hc.Privileged = c.Privileged

	cont, err := d.c.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		utils.FQDN(c.Name, networkName),
	)
	if err != nil {
		return "", err
	}

	err = d.c.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	return cont.ID, nil
}

// PullImage pulls a Docker image from a remote repo
func (d *DockerTasks) PullImage(image config.Image, force bool) error {
	in := makeImageCanonical(image.Name)

	args := filters.NewArgs()
	args.Add("reference", image.Name)

	// only pull if image is not in current registry so check to see if the image is present
	// if force then skil this check
	if !force {
		sum, err := d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
		if err != nil {
			return xerrors.Errorf("unable to list images in local Docker cache: %w", err)
		}

		// if we have images do not pull
		if len(sum) > 0 {
			d.l.Debug("Image exists in local cache", "image", image.Name)

			return nil
		}
	}

	ipo := types.ImagePullOptions{}

	// if the username and password is not null make an authenticated
	// image pull
	if image.Username != "" && image.Password != "" {
		ipo.RegistryAuth = createRegistryAuth(image.Username, image.Password)
	}

	d.l.Debug("Pulling image", "image", image.Name)

	out, err := d.c.ImagePull(context.Background(), in, ipo)
	if err != nil {
		return xerrors.Errorf("Error pulling image: %w", err)
	}

	// write the output to /dev/null
	// TODO this stuff needs to be logged correctly
	io.Copy(ioutil.Discard, out)

	return nil
}

// FindContainerIDs returns the Container IDs for the given identifier
func (d *DockerTasks) FindContainerIDs(containerName string, networkName string) ([]string, error) {
	fullName := utils.FQDN(containerName, networkName)

	args := filters.NewArgs()
	args.Add("name", fullName)

	opts := types.ContainerListOptions{Filters: args, All: true}

	cl, err := d.c.ContainerList(context.Background(), opts)
	if err != nil || cl == nil {
		return nil, err
	}

	if len(cl) > 0 {
		ids := []string{}
		for _, c := range cl {
			ids = append(ids, c.ID)
		}

		return ids, nil
	}

	return nil, nil
}

// RemoveContainer with the given id
func (d *DockerTasks) RemoveContainer(id string) error {
	return d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}

// CreateVolume creates a Docker volume for a cluster
// returns the volume name and an error if unsuccessful
func (d *DockerTasks) CreateVolume(name string) (string, error) {
	vn := volumeName(name)
	d.l.Debug("Create Volume", "ref", name, "name", vn)

	volumeCreateOptions := volume.VolumeCreateBody{
		Name:       vn,
		Driver:     "local", //TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	vol, err := d.c.VolumeCreate(context.Background(), volumeCreateOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create image volume [%s] for cluster [%s]\n%+v", vn, name, err)
	}

	return vol.Name, nil
}

// RemoveVolume deletes the Docker volume associated with  a cluster
func (d *DockerTasks) RemoveVolume(name string) error {
	vn := volumeName(name)
	d.l.Debug("Deleting Volume", "ref", name, "name", vn)

	return d.c.VolumeRemove(context.Background(), vn, true)
}

// ContainerLogs streams the logs for the container to the returned io.ReadCloser
func (d *DockerTasks) ContainerLogs(id string, stdOut, stdErr bool) (io.ReadCloser, error) {
	return d.c.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{ShowStderr: stdErr, ShowStdout: stdOut})
}

// CopyFromContainer copies a file from a container
func (d *DockerTasks) CopyFromContainer(id, src, dst string) error {
	d.l.Debug("Copying file from", "id", id, "src", src, "dst", dst)

	reader, _, err := d.c.CopyFromContainer(context.Background(), id, src)
	if err != nil {
		return fmt.Errorf("Couldn't copy kubeconfig.yaml from server container %s\n%+v", id, err)
	}
	defer reader.Close()

	readBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("Couldn't read kubeconfig from container\n%+v", err)
	}

	// write to file, skipping the first 512 bytes which contain file metadata
	// and trimming any NULL characters
	trimBytes := bytes.Trim(readBytes[512:], "\x00")

	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Couldn't create file %s\n%+v", dst, err)
	}

	defer file.Close()
	file.Write(trimBytes)

	return nil
}

func volumeName(clusterName string) string {
	return fmt.Sprintf("%s.volume", clusterName)
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

// credentials are a json string and need to be base64 encoded
func createRegistryAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(
			fmt.Sprintf(`{"Username": "%s", "Password": "%s"}`, username, password),
		),
	)
}

// makeImageCanonical makes sure the image reference uses full canonical name i.e.
// consul:1.6.1 -> docker.io/library/consul:1.6.1
func makeImageCanonical(image string) string {
	imageParts := strings.Split(image, "/")
	switch len(imageParts) {
	case 1:
		return fmt.Sprintf("docker.io/library/%s", imageParts[0])
	case 2:
		return fmt.Sprintf("docker.io/%s/%s", imageParts[0], imageParts[1])
	}

	return image
}
