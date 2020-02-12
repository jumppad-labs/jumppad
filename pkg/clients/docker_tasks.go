package clients

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

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
func (d *DockerTasks) CreateContainer(c *config.Container) (string, error) {
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
		utils.FQDN(c.Name, string(c.Type)),
	)
	if err != nil {
		return "", err
	}

	for _, n := range c.Networks {
		net, err := c.FindDependentResource(n.Name)
		if err != nil {
			return "", xerrors.Errorf("Network not found: %w", err)
		}

		d.l.Debug("Attaching container to network", "ref", c.Name, "network", n.Name)
		es := &network.EndpointSettings{NetworkID: net.Info().Name}

		// are we binding to a specific ip
		if n.IPAddress != "" {
			d.l.Debug("Assigning static ip address", "ref", c.Name, "network", n.Name, "ip_address", n.IPAddress)
			es.IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: n.IPAddress}
		}

		err = d.c.NetworkConnect(context.Background(), net.Info().Name, cont.ID, es)
		if err != nil {
			// if we fail to connect to the network roll back the container
			errRemove := d.RemoveContainer(cont.ID)
			if errRemove != nil {
				return "", xerrors.Errorf("Unable to connect container to network %s, unable to roll back container: %w", n.Name, err)
			}

			return "", xerrors.Errorf("Unable to connect container to network %s: %w", n.Name, err)
		}
	}

	// attach the container to the network
	// if c.Network != nil {
	// 	d.l.Debug("Attaching container to network", "ref", c.Name, "network", c.NetworkRef.Name)
	// 	es := &network.EndpointSettings{NetworkID: c.NetworkRef.Name}

	// 	// are we binding to a specific ip
	// 	if c.IPAddress != "" {
	// 		d.l.Debug("Assigning static ip address", "ref", c.Name, "network", c.NetworkRef.Name, "ip_address", c.IPAddress)
	// 		es.IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: c.IPAddress}
	// 	}

	// 	err := d.c.NetworkConnect(context.Background(), c.NetworkRef.Name, cont.ID, es)
	// 	if err != nil {
	// 		// if we fail to connect to the network roll back the container
	// 		errRemove := d.RemoveContainer(cont.ID)
	// 		if errRemove != nil {
	// 			return "", xerrors.Errorf("Unable to connect container to network %s, unable to roll back container: %w", c.NetworkRef.Name, err)
	// 		}

	// 		return "", xerrors.Errorf("Unable to connect container to network %s: %w", c.NetworkRef.Name, err)
	// 	}
	// }

	// attach the container to the WAN network
	// if c.WANRef != nil {
	// 	d.l.Debug("Attaching container to WAN network", "ref", c.Name, "network", c.WANRef.Name)
	// 	es := &network.EndpointSettings{NetworkID: c.WANRef.Name}

	// 	err := d.c.NetworkConnect(context.Background(), c.WANRef.Name, cont.ID, es)
	// 	if err != nil {
	// 		errRemove := d.RemoveContainer(cont.ID)
	// 		if errRemove != nil {
	// 			return "", xerrors.Errorf("Unable to connect container to wan network %s, unable to roll back container: %w", c.WANRef.Name, err)
	// 		}

	// 		return "", xerrors.Errorf("Unable to connect container to wan network %s: %w", c.WANRef.Name, err)
	// 	}
	// }

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
	vn := utils.FQDNVolumeName(name)
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
	vn := utils.FQDNVolumeName(name)
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

// CopyLocalDockerImageToVolume writes multiple Docker images to a Docker volume as a compressed archive
// returns the filename of the archive and an error if one occured
func (d *DockerTasks) CopyLocalDockerImageToVolume(images []string, volume string) (string, error) {
	d.l.Debug("Writing docker images to volume", "images", images, "volume", volume)

	// save the image to a local temp file
	ir, err := d.c.ImageSave(context.Background(), images)
	if err != nil {
		return "", xerrors.Errorf("unable to save images: %w", err)
	}
	defer ir.Close()

	// create a temp file to hold the tar
	tmpFile, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		return "", xerrors.Errorf("unable to create temporary file: %w", err)
	}

	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, ir)
	if err != nil {
		return "", xerrors.Errorf("unable to copy image to temp file: %w", err)
	}

	// set the seek pos back to 0
	tmpFile.Seek(0, 0)

	// create  temp file for a tar to contain the tar we just created
	// CopyToContainer expects a tar which has individual file entries
	// if we write the original file the output will not be a single file
	// but the contents of the tar. To bypass this we need to add the output from
	// save image to a tar
	tmpTarFile, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		return "", xerrors.Errorf("unable to create temporary file: %w", err)
	}

	defer tmpTarFile.Close()

	_, err = io.Copy(tmpTarFile, ir)
	if err != nil {
		return "", xerrors.Errorf("unable to copy image to temp file: %w", err)
	}

	ta := tar.NewWriter(tmpTarFile)

	fi, _ := tmpFile.Stat()

	hdr, err := tar.FileInfoHeader(fi, fi.Name())
	if err != nil {
		return "", xerrors.Errorf("unable to create header for tar: %w", err)
	}

	// write the header to the tar file, this has to happen before the file
	err = ta.WriteHeader(hdr)
	if err != nil {
		return "", xerrors.Errorf("unable to write tar header: %w", err)
	}

	io.Copy(ta, tmpFile)
	if err != nil {
		return "", xerrors.Errorf("unable to copy image to tar file: %w", err)
	}

	ta.Close()

	// reset the file seek so we can copy to the container
	tmpTarFile.Seek(0, 0)

	// create a dummy container to import to volume
	cc := config.NewContainer("temp-import")

	cc.Image = config.Image{Name: makeImageCanonical("alpine:latest")}
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volume,
			Destination: "/images",
			Type:        "volume",
		},
	}
	cc.Command = []string{"tail", "-f", "/dev/null"}

	tmpID, err := d.CreateContainer(cc)
	if err != nil {
		return "", xerrors.Errorf("unable to create dummy container for importing images: %w", err)
	}
	defer d.RemoveContainer(tmpID)

	err = d.c.CopyToContainer(context.Background(), utils.FQDN(cc.Name, string(cc.Type)), "/images", tmpTarFile, types.CopyToContainerOptions{})
	if err != nil {
		return "", xerrors.Errorf("unable to copy file to container: %w", err)
	}

	// return the name of the archive
	return fi.Name(), nil
}

// ExecuteCommand allows the execution of commands in a running docker container
// id is the id of the container to execute the command in
// command is a slice of strings to execute
// writer [optional] will be used to write any output from the command execution.
func (d *DockerTasks) ExecuteCommand(id string, command []string, writer io.Writer) error {
	execid, err := d.c.ContainerExecCreate(context.Background(), id, types.ExecConfig{
		Cmd:          command,
		WorkingDir:   "/",
		AttachStdout: true,
		AttachStderr: true,
	})

	if err != nil {
		return xerrors.Errorf("unable to create container exec: %w", err)
	}

	// get logs from an attach
	stream, err := d.c.ContainerExecAttach(context.Background(), execid.ID, types.ExecStartCheck{})
	if err != nil {
		return xerrors.Errorf("unable to attach logging to exec process: %w", err)
	}
	defer stream.Close()

	// ensure that the log from the Docker exec command is copied to the default logger
	if writer != nil {
		go func() {
			io.Copy(
				writer,
				stream.Reader,
			)
		}()
	}

	err = d.c.ContainerExecStart(context.Background(), execid.ID, types.ExecStartCheck{})
	if err != nil {
		return xerrors.Errorf("unable to start exec process: %w", err)
	}

	// loop until the container finishes execution
	for {
		i, err := d.c.ContainerExecInspect(context.Background(), execid.ID)
		if err != nil {
			return xerrors.Errorf("unable to determine status of exec process: %w", err)
		}

		if !i.Running {
			if i.ExitCode == 0 {
				return nil
			}

			return xerrors.Errorf("container exec failed with exit code %d", i.ExitCode)
		}

		time.Sleep(1 * time.Second)
	}
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
