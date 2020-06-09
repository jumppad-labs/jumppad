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
	gosignal "os/signal"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/signal"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/streams"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// DockerTasks is a concrete implementation of ContainerTasks which uses the Docker SDK
type DockerTasks struct {
	c     Docker
	il    ImageLog
	force bool
	l     hclog.Logger
}

// NewDockerTasks creates a DockerTasks with the given Docker client
func NewDockerTasks(c Docker, il ImageLog, l hclog.Logger) *DockerTasks {
	return &DockerTasks{c: c, il: il, l: l}
}

// SetForcePull sets a global override for the DockerTasks, when set to true
// Images will always be pulled from remote registries
func (d *DockerTasks) SetForcePull(force bool) {
	d.force = force
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
		Entrypoint:   c.Entrypoint,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	// create the host and network configs
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	// by default the container should NOT be attached to a network
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

		// if we have a bind type mount then ensure that the local folder exists or
		// an error will be raised when creating
		if t == mount.TypeBind {
			// check to see id the source exists
			_, err := os.Stat(vc.Source)
			if err != nil {
				d.l.Debug("Creating directory for container volume", "ref", c.Name, "directory", vc.Source, "volume", vc.Destination)
				// source does not exist, create the source as a directory
				err := os.MkdirAll(vc.Source, os.ModePerm)
				if err != nil {
					return "", xerrors.Errorf("Source for Volume %s does not exist, error creating directory: %w", err)
				}
			}
		}

		// create the mount
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

	// are we attaching the container to a sidecar network?
	for _, n := range c.Networks {
		net, err := c.FindDependentResource(n.Name)
		if err != nil {
			return "", xerrors.Errorf("Network not found: %w", err)
		}

		if net.Info().Type == config.TypeContainer {
			// find the id of the container
			ids, err := d.FindContainerIDs(net.Info().Name, net.Info().Type)
			if err != nil {
				return "", xerrors.Errorf("Unable to attach to container network, ID for container not found: %w", err)
			}

			if len(ids) != 1 {
				return "", xerrors.Errorf("Unable to attach to container network, ID for container not found")
			}

			d.l.Debug("Attaching container as sidecar", "ref", c.Name, "container", n.Name)

			// set the container network
			hc.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%s", ids[0]))
			// when using container networking can not use a hostname
			dc.Hostname = ""
		}
	}

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

	// first remove the container from the bridge network if we are adding custom networks
	// all containers should have custom networks
	// only add networks if we are not adding the container network
	if len(c.Networks) > 0 && !hc.NetworkMode.IsContainer() {
		err := d.c.NetworkDisconnect(context.Background(), "bridge", cont.ID, true)
		if err != nil {
			return "", xerrors.Errorf("Unable to remove container from the default bridge network: %w", err)
		}

		for _, n := range c.Networks {
			net, err := c.FindDependentResource(n.Name)
			if err != nil {
				errRemove := d.RemoveContainer(cont.ID)
				if errRemove != nil {
					return "", xerrors.Errorf("Unable to connect container to network %s, unable to roll back container: %w", n.Name, err)
				}

				return "", xerrors.Errorf("Network not found: %w", err)
			}

			d.l.Debug("Attaching container to network", "ref", c.Name, "network", n.Name)
			es := &network.EndpointSettings{NetworkID: net.Info().Name}

			// if we have network aliases defined, add them to the network connection
			if n.Aliases != nil && len(n.Aliases) > 0 {
				es.Aliases = n.Aliases
			}

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
	if !force && !d.force {
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

	// update the image log
	err = d.il.Log(in, ImageTypeDocker)
	if err != nil {
		d.l.Error("Unable to add image name to cache", "error", err)
	}

	// write the output to /dev/null
	// TODO this stuff needs to be logged correctly
	io.Copy(ioutil.Discard, out)

	return nil
}

// FindContainerIDs returns the Container IDs for the given identifier
func (d *DockerTasks) FindContainerIDs(containerName string, typeName config.ResourceType) ([]string, error) {
	fullName := utils.FQDN(containerName, string(typeName))

	args := filters.NewArgs()
	// By default Docker will wildcard searches, use regex to return the absolute
	args.Add("name", fmt.Sprintf("^/%s$", fullName))

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
	// try and shutdown graceful
	err := d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: false, RemoveVolumes: true})

	// unable to shutdown graceful try force
	if err != nil {
		return d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	}

	return nil
}

// CreateVolume creates a Docker volume for a cluster
// if the volume exists performs no action
// returns the volume name and an error if unsuccessful
func (d *DockerTasks) CreateVolume(name string) (string, error) {
	vn := utils.FQDNVolumeName(name)

	args := filters.NewArgs()
	// By default Docker will wildcard searches, use regex to return the absolute
	args.Add("name", vn)
	ops, err := d.c.VolumeList(context.Background(), args)
	if err != nil {
		return "", fmt.Errorf("unable to lookup volume [%s] for cluster [%s]\n%+v", vn, name, err)
	}

	if len(ops.Volumes) > 0 {
		d.l.Debug("Volume exists", "ref", name, "name", vn)
		return vn, nil
	}

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
func (d *DockerTasks) CopyLocalDockerImageToVolume(images []string, volume string, force bool) ([]string, error) {
	d.l.Debug("Writing docker images to volume", "images", images, "volume", volume)

	savedImages := []string{}

	// make sure we have the alpine image needed to copy
	err := d.PullImage(config.Image{Name: "alpine:latest"}, false)
	if err != nil {
		return nil, xerrors.Errorf("Unable pull alpine:latest for importing images: %w", err)
	}

	// create a dummy container to import to volume
	cc := config.NewContainer("temp-import")

	cc.Image = config.Image{Name: "alpine:latest"}
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
		return nil, xerrors.Errorf("unable to create dummy container for importing images: %w", err)
	}
	defer d.RemoveContainer(tmpID)

	// add each image individually
	for _, i := range images {
		compressedImageName := fmt.Sprintf("%s.tar", base64.StdEncoding.EncodeToString([]byte(i)))

		// check if the image exists if we are not doing a forced update
		if !d.force && !force {
			err := d.ExecuteCommand(tmpID, []string{"find", "/images/" + compressedImageName}, nil, "/", nil)
			if err == nil {
				// we have the image already
				d.l.Debug("Image already cached", "image", i)
				savedImages = append(savedImages, compressedImageName)
				continue
			}
		}

		d.l.Debug("Copying image to container", "image", i)

		// save the image to a local temp file
		ir, err := d.c.ImageSave(context.Background(), []string{i})
		if err != nil {
			return nil, xerrors.Errorf("unable to save images: %w", err)
		}
		defer ir.Close()

		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, xerrors.Errorf("unable to create temporary file: %w", err)
		}

		// create a temp file to hold the tar
		tmpFile, err := os.Create(path.Join(tmpDir, compressedImageName))
		if err != nil {
			return nil, xerrors.Errorf("unable to create temporary file: %w", err)
		}

		defer func() {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}()

		_, err = io.Copy(tmpFile, ir)
		if err != nil {
			return nil, xerrors.Errorf("unable to copy image to temp file: %w", err)
		}

		// set the seek pos back to 0
		tmpFile.Seek(0, 0)

		// create  temp file for a tar to contain the tar we just created
		// CopyToContainer expects a tar which has individual file entries
		// if we write the original file the output will not be a single file
		// but the contents of the tar. To bypass this we need to add the output from
		// save image to a tar
		tmpTarFile, err := ioutil.TempFile("", "")
		if err != nil {
			return nil, xerrors.Errorf("unable to create temporary file: %w", err)
		}

		defer func() {
			tmpTarFile.Close()
			os.Remove(tmpTarFile.Name())
		}()

		_, err = io.Copy(tmpTarFile, ir)
		if err != nil {
			return nil, xerrors.Errorf("unable to copy image to temp file: %w", err)
		}

		ta := tar.NewWriter(tmpTarFile)

		fi, _ := tmpFile.Stat()

		hdr, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return nil, xerrors.Errorf("unable to create header for tar: %w", err)
		}

		// write the header to the tar file, this has to happen before the file
		err = ta.WriteHeader(hdr)
		if err != nil {
			return nil, xerrors.Errorf("unable to write tar header: %w", err)
		}

		io.Copy(ta, tmpFile)
		if err != nil {
			return nil, xerrors.Errorf("unable to copy image to tar file: %w", err)
		}

		ta.Close()

		// reset the file seek so we can copy to the container
		tmpTarFile.Seek(0, 0)

		err = d.c.CopyToContainer(context.Background(), utils.FQDN(cc.Name, string(cc.Type)), "/images", tmpTarFile, types.CopyToContainerOptions{})
		if err != nil {
			return nil, xerrors.Errorf("unable to copy file to container: %w", err)
		}

		savedImages = append(savedImages, compressedImageName)
	}

	// return the name of the archive
	return savedImages, nil
}

// ExecuteCommand allows the execution of commands in a running docker container
// id is the id of the container to execute the command in
// command is a slice of strings to execute
// writer [optional] will be used to write any output from the command execution.
func (d *DockerTasks) ExecuteCommand(id string, command []string, env []string, workingDir string, writer io.Writer) error {
	execid, err := d.c.ContainerExecCreate(context.Background(), id, types.ExecConfig{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		Env:          env,
		WorkingDir:   workingDir,
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

	// if we have a writer stream the logs from the container to the writer
	if writer != nil {

		ttyOut := streams.NewOut(writer)
		ttyErr := streams.NewOut(writer)

		errCh := make(chan error, 1)

		go func() {
			defer close(errCh)
			errCh <- func() error {

				streamer := streams.NewHijackedStreamer(nil, ttyOut, nil, ttyOut, ttyErr, stream, false, "", d.l)

				return streamer.Stream(context.Background())
			}()
		}()

		/*
			if err := <-errCh; err != nil {
				d.l.Error("unable to hijack exec stream: %s", err)
				return err
			}
		*/
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

// TODO: this is all exploritory, works but needs a major tidy

// CreateShell creates an interactive shell inside a container
// https://github.com/docker/cli/blob/ae1618713f83e7da07317d579d0675f578de22fa/cli/command/container/exec.go
func (d *DockerTasks) CreateShell(id string, command []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error {
	execid, err := d.c.ContainerExecCreate(context.Background(), id, types.ExecConfig{
		Cmd:          command,
		WorkingDir:   "/",
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	})

	if err != nil {
		return xerrors.Errorf("unable to create container exec: %w", err)
	}

	// err = d.c.ContainerExecStart(context.Background(), execid.ID, types.ExecStartCheck{})
	// if err != nil {
	// 	return xerrors.Errorf("unable to start exec process: %w", err)
	// }

	resp, err := d.c.ContainerExecAttach(context.Background(), execid.ID, types.ExecStartCheck{Tty: true})
	if err != nil {
		return err
	}

	// wrap the standard streams
	ttyIn := streams.NewIn(stdin)
	ttyOut := streams.NewOut(stdout)
	ttyErr := streams.NewOut(stderr)

	defer resp.Close()

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		errCh <- func() error {

			streamer := streams.NewHijackedStreamer(ttyIn, ttyOut, ttyIn, ttyOut, ttyErr, resp, true, "", d.l)

			return streamer.Stream(context.Background())
		}()
	}()

	// init the TTY
	d.initTTY(execid.ID, ttyOut)

	// monitor for TTY changes
	sigchan := make(chan os.Signal, 1)
	gosignal.Notify(sigchan, signal.SIGWINCH)
	go func() {
		for range sigchan {
			d.resizeTTY(execid.ID, ttyOut)
		}
	}()

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

func (d *DockerTasks) initTTY(id string, out *streams.Out) error {
	if err := d.resizeTTY(id, out); err != nil {
		go func() {
			var err error
			for retry := 0; retry < 5; retry++ {
				time.Sleep(10 * time.Millisecond)
				if err = d.resizeTTY(id, out); err == nil {
					break
				}
			}
			if err != nil {
				//something
				d.l.Error("Unable to resize TTY use default", "error", err)
			}
		}()
	}

	return nil
}

func (d *DockerTasks) resizeTTY(id string, out *streams.Out) error {
	h, w := out.GetTtySize()

	if h == 0 && w == 0 {
		return nil
	}

	options := types.ResizeOptions{
		Height: uint(h),
		Width:  uint(w),
	}

	// resize the contiainer
	err := d.c.ContainerExecResize(context.Background(), id, options)
	if err != nil {
		return err
	}

	return nil
}

// DetachNetwork detaches a container from a network
// TODO: Docker returns success before removing a container
// tasks which depend on the network being removed may fail in the future
// we need to check it has been removed before returning
func (d *DockerTasks) DetachNetwork(network, containerid string) error {
	network = strings.Replace(network, "network.", "", -1)
	err := d.c.NetworkDisconnect(context.Background(), network, containerid, true)

	// Hacky hack for now
	//time.Sleep(1000 * time.Millisecond)

	return err
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
		dp, _ := nat.NewPort(p.Protocol, p.Local)
		pp.ExposedPorts[dp] = struct{}{}

		pb := []nat.PortBinding{
			nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: p.Host,
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
