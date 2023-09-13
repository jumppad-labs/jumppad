package container

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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	dtypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/images"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/streams"
	ctar "github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/moby/sys/signal"
	"github.com/moby/term"
	"golang.org/x/xerrors"
)

const (
	StorageDriverOverlay2     = "overlay2"
	StorageDriverFuse         = "fuse-overlayfs"
	StorageDriverBTRFS        = "btrfs"
	StorageDriverZFS          = "zfs"
	StorageDriverVFS          = "vfs"
	StorageDriverAUFS         = "aufs"
	StorageDriverDeviceMapper = "devicemapper"
	StorageDriverOverlay      = "overlay"
)

const defaultExitCode = 254

// DockerTasks is a concrete implementation of ContainerTasks which uses the Docker SDK
type DockerTasks struct {
	engineType    string
	storageDriver string
	c             Docker
	il            images.ImageLog
	l             logger.Logger
	tg            *ctar.TarGz
	force         bool
}

// NewDockerTasks creates a DockerTasks with the given Docker client
func NewDockerTasks(c Docker, il images.ImageLog, tg *ctar.TarGz, l logger.Logger) (*DockerTasks, error) {

	// Set the engine type, Docker, Podman
	ver, err := c.ServerVersion(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Error checking server version: %s", err)
	}

	t := dtypes.EngineNotFound

	for _, c := range ver.Components {
		switch c.Name {
		case "Engine":
			t = dtypes.EngineTypeDocker
		case "Podman Engine":
			t = dtypes.EngineTypePodman
		}
	}

	// Determine the storage driver
	info, err := c.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Error checking storage driver: %s", err)
	}

	return &DockerTasks{engineType: t, storageDriver: info.Driver, c: c, il: il, tg: tg, l: l}, nil
}

func (d *DockerTasks) EngineInfo() *dtypes.EngineInfo {
	return &dtypes.EngineInfo{StorageDriver: d.storageDriver, EngineType: d.engineType}
}

// SetForcePull sets a global override for the DockerTasks, when set to true
// Images will always be pulled from remote registries
func (d *DockerTasks) SetForcePull(force bool) {
	d.force = force
}

// CreateContainer creates a new Docker container for the given configuation
func (d *DockerTasks) CreateContainer(c *dtypes.Container) (string, error) {
	d.l.Debug("Creating Docker Container", "ref", c.Name)

	// ensure the image name is the full canonical image as Podman does not use the
	// default docker.io registry
	c.Image.Name = makeImageCanonical(c.Image.Name)

	// create a unique name based on service network [container].[network].shipyard
	// attach to networks
	// - networkRef
	// - wanRef

	// convert the environment vars to a list of [key]=[value]
	env := make([]string, 0)
	for k, v := range c.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// set the user details
	var user string
	if c.RunAs != nil {
		user = fmt.Sprintf("%s:%s", c.RunAs.User, c.RunAs.Group)
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
		User:         user,
	}

	// create the host and network configs
	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	// add any dns servers
	hc.DNS = c.DNS

	if c.MaxRestartCount > 0 {
		hc.RestartPolicy = container.RestartPolicy{Name: "on-failure", MaximumRetryCount: c.MaxRestartCount}
	}

	// https: //docs.docker.com/config/containers/resource_constraints/#cpu
	rc := container.Resources{}
	if c.Resources != nil {
		// set memory if set
		if c.Resources.Memory > 0 {
			rc.Memory = int64(c.Resources.Memory) * 1000000 // docker specifies memory in bytes, shipyard megabytes
		}

		if c.Resources.CPU > 0 {
			rc.CPUQuota = int64(c.Resources.CPU) * 100
		}

		// cupsets are not supported on windows
		if len(c.Resources.CPUPin) > 0 {
			cpuPin := make([]string, len(c.Resources.CPUPin))
			for i, v := range c.Resources.CPUPin {
				cpuPin[i] = fmt.Sprintf("%d", v)
			}

			rc.CpusetCpus = strings.Join(cpuPin, ",")
		}

		hc.Resources = rc
	}

	// by default the container should NOT be attached to a network
	nc.EndpointsConfig = make(map[string]*network.EndpointSettings)

	// Create mounts
	mounts := make([]mount.Mount, 0)
	volumes := []string{}

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

		bp := mount.PropagationRPrivate
		switch vc.BindPropagation {
		case "shared":
			bp = mount.PropagationShared
		case "slave":
			bp = mount.PropagationSlave
		case "private":
			bp = mount.PropagationPrivate
		case "rslave":
			bp = mount.PropagationRSlave
		case "rprivate":
			bp = mount.PropagationRPrivate
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
					return "", xerrors.Errorf("source for Volume %s does not exist, error creating directory: %w", err)
				}
			}
		}

		var bindOptions *mount.BindOptions
		if t == mount.TypeBind {
			bindOptions = &mount.BindOptions{Propagation: bp, NonRecursive: vc.BindPropagationNonRecursive}
		}

		// Volumes in podman are mounted read only by default, we need to add the :z parameter to
		// ensure that the correct selinux flags are set so that we can write to these volumes
		if t == mount.TypeVolume {
			readOnly := ""
			if vc.ReadOnly {
				readOnly = ":ro"
			}

			volumes = append(volumes, fmt.Sprintf("%s:%s:z%s", vc.Source, vc.Destination, readOnly))
			continue
		}

		// create the mount
		mounts = append(mounts, mount.Mount{
			Type:        t,
			Source:      vc.Source,
			Target:      vc.Destination,
			ReadOnly:    vc.ReadOnly,
			BindOptions: bindOptions,
		})
	}

	hc.Mounts = mounts
	hc.Binds = volumes

	// create the ports config
	ports := createPublishedPorts(c.Ports)
	dc.ExposedPorts = ports.ExposedPorts
	hc.PortBindings = ports.PortBindings

	// create the port ranges
	portRanges, err := createPublishedPortRanges(c.PortRanges)
	if err != nil {
		return "", xerrors.Errorf("unable to attach to container network, invalid port range: %w", err)
	}

	for k, p := range portRanges.ExposedPorts {
		// check that the port has not already been defined
		if _, ok := dc.ExposedPorts[k]; !ok {
			dc.ExposedPorts[k] = p
			hc.PortBindings[k] = portRanges.PortBindings[k]
		}
	}

	// is this a priviledged container
	hc.Privileged = c.Privileged

	// are we attaching the container to a sidecar network?
	for _, n := range c.Networks {
		if n.IsContainer {
			d.l.Debug("Attaching as sidecar", "ref", c.Name, "container", n.ID)

			// set the container network
			hc.NetworkMode = container.NetworkMode(fmt.Sprintf("container:%s", n.ID))
			// when using container networking can not use a hostname
			dc.Hostname = ""
		}
	}

	cont, err := d.c.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		nil,
		c.Name,
	)
	if err != nil {
		return "", err
	}

	// first remove the container from the default network if we are adding custom networks
	if len(c.Networks) > 0 && !hc.NetworkMode.IsContainer() {
		d.l.Debug("Remove container from default networks", "ref", c.Name)

		info, err := d.c.ContainerInspect(context.Background(), cont.ID)
		if err != nil {
			return "", xerrors.Errorf("unable to remove container from the default network: %w", err)
		}

		// get all default attached networks, we will disconnect these later
		defaultNets := []string{}
		for k, _ := range info.NetworkSettings.Networks {
			defaultNets = append(defaultNets, k)
		}

		// attach the custom networks
		for _, n := range c.Networks {
			net, err := d.FindNetwork(n.ID)
			if err != nil {
				return "", err
			}

			err = d.AttachNetwork(net.Name, cont.ID, n.Aliases, n.IPAddress)

			if err != nil {
				// if we fail to connect to the network roll back the container
				errRemove := d.RemoveContainer(cont.ID, false)
				if errRemove != nil {
					return "", xerrors.Errorf("failed to attach network %s to container %s, unable to roll back container: %w", n.ID, cont.ID, err)
				}

				return "", xerrors.Errorf("unable to connect container to network %s, successfully rolled back container: %w", n.ID, err)
			}
		}

		// disconnect the default networks
		// for podman this needs to happen after we have attached to a network or it fails silently
		for _, n := range defaultNets {
			d.l.Debug("Disconnecting network", "name", n, "ref", c.Name)

			err := d.c.NetworkDisconnect(context.Background(), n, cont.ID, true)
			if err != nil {
				d.l.Warn("Unable to remove container from the network", "name", n, "ref", c.Name, "error", err)
			}
		}
	}

	err = d.c.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	return cont.ID, nil
}

// ContainerInfo returns the Docker container info
func (d *DockerTasks) ContainerInfo(id string) (interface{}, error) {
	cj, err := d.c.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, xerrors.Errorf("unable to read information about Docker container %s: %w", id, err)
	}

	return cj, nil
}

// FindImageInLocalRegistry returns the id for an image in the local registry that
// matches the given tag. If no image is found an empty string is returned.
func (d *DockerTasks) FindImageInLocalRegistry(image dtypes.Image) (string, error) {
	images, err := d.FindImagesInLocalRegistry(image.Name)
	if err != nil {
		return "", err
	}

	if len(images) > 0 {
		return images[0], nil
	}

	return "", nil
}

// Finds images in the local registry that match the filter
func (d *DockerTasks) FindImagesInLocalRegistry(filter string) ([]string, error) {
	images := []string{}

	args := filters.NewArgs()
	args.Add("reference", filter)

	sum, err := d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
	if err != nil {
		return nil, xerrors.Errorf("unable to list images in local Docker cache: %w", err)
	}

	// we have images return
	if len(sum) > 0 {
		for _, i := range sum {
			images = append(images, i.ID)
		}

		return images, nil
	}

	// check the canonical name as this might be a podman docker server
	in := makeImageCanonical(filter)
	args = filters.NewArgs()
	args.Add("reference", in)

	sum, err = d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
	if err != nil {
		return nil, xerrors.Errorf("unable to list images in local Docker cache: %w", err)
	}

	// we have images do not pull
	if len(sum) > 0 {
		for _, i := range sum {
			images = append(images, i.ID)
		}

		return images, nil
	}

	return nil, nil
}

// PullImage pulls a Docker image from a remote repo
func (d *DockerTasks) PullImage(image dtypes.Image, force bool) error {
	// if image is local, do not try to pull jumppad.dev/localcache
	if strings.HasPrefix(image.Name, utils.BuildImagePrefix) {
		return nil
	}

	in := makeImageCanonical(image.Name)

	// only pull if image is not in current registry so check to see if the image is present
	// if force then skip this check
	if !force && !d.force {
		id, err := d.FindImageInLocalRegistry(image)
		if err != nil {
			return err
		}

		// found the image do nothing
		if id != "" {
			return nil
		}
	}

	ipo := types.ImagePullOptions{}

	// if the username and password is not null make an authenticated
	// image pull
	if image.Username != "" && image.Password != "" {
		ipo.RegistryAuth = createRegistryAuth(image.Username, image.Password)
	}

	d.l.Debug("Pulling image", "image", in)

	out, err := d.c.ImagePull(context.Background(), in, ipo)
	if err != nil {
		return xerrors.Errorf("Error pulling image: %w", err)
	}

	// update the image log
	err = d.il.Log(in, images.ImageTypeDocker)
	if err != nil {
		d.l.Error("Unable to add image name to cache", "error", err)
	}

	// write the output to the debug log
	io.Copy(d.l.StandardWriter(), out)

	return nil
}

// FindContainerIDs returns the Container IDs for the given identifier
func (d *DockerTasks) FindContainerIDs(fqdn string) ([]string, error) {
	args := filters.NewArgs()
	// By default Docker will wildcard searches, use regex to return the absolute
	args.Add("name", fmt.Sprintf("^%s$", fqdn))

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
func (d *DockerTasks) RemoveContainer(id string, force bool) error {
	var err error
	if !force {
		// try and shutdown graceful
		timeout := 30
		err = d.c.ContainerStop(context.Background(), id, container.StopOptions{Timeout: &timeout})
		if err == nil {
			d.l.Debug("Container stopped gracefully, removing", "container", id)

			err = d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: false, RemoveVolumes: true})
			if err == nil {
				return nil
			}
		}

		d.l.Debug("Unable to stop container gracefully, trying force", "container", id, "error", err)
	}

	// unable to shutdown graceful try force
	d.l.Debug("Forcefully remove", "container", id)
	return d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}

func (d *DockerTasks) RemoveImage(id string) error {
	_, err := d.c.ImageRemove(context.Background(), id, types.ImageRemoveOptions{Force: true})

	return err
}

func (d *DockerTasks) BuildContainer(config *dtypes.Build, force bool) (string, error) {
	// get the checksum for the id
	cs, err := utils.HashDir(config.Context)
	if err != nil {
		return "", err
	}

	// strip the prefix and grab the first 8chars for the id
	cs, _ = utils.ReplaceNonURIChars(cs[3:11])

	// create the fully qualified name using the checksum for the content
	imageWithId := fmt.Sprintf("jumppad.dev/localcache/%s:%s", config.Name, strings.ToLower(cs))
	imageWithId = makeImageCanonical(imageWithId)

	args := filters.NewArgs()
	args.Add("reference", imageWithId)

	// check if the image already exists, if so do not rebuild unless force
	if !force && !d.force {
		sum, err := d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
		if err != nil {
			return "", xerrors.Errorf("unable to list images in local Docker cache: %w", err)
		}

		// if we have images do not build
		if len(sum) > 0 {
			d.l.Debug("Image exists in local cache, skip build", "image", imageWithId)

			return imageWithId, nil
		}
	}

	// if the Dockerfile is not set, set to default
	if config.DockerFile == "" {
		config.DockerFile = "./Dockerfile"
	}

	// configure the build args
	buildArgs := map[string]*string{}
	for k, v := range config.Args {
		buildArgs[k] = &v
	}

	d.l.Debug("Building image", "id", imageWithId)

	// tar the build context folder and send to the server
	buildOpts := types.ImageBuildOptions{
		Dockerfile: config.DockerFile,
		Tags:       []string{imageWithId},
		Remove:     true,
		BuildArgs:  buildArgs,
	}

	var buf bytes.Buffer
	d.tg.Compress(&buf, &ctar.TarGzOptions{OmitRoot: true}, config.Context)

	resp, err := d.c.ImageBuild(context.Background(), &buf, buildOpts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out := d.l.StandardWriter()
	termFd, _ := term.GetFdInfo(out)
	err = jsonmessage.DisplayJSONMessagesStream(resp.Body, out, termFd, false, nil)

	if err != nil {
		return "", err
	}

	return imageWithId, nil
}

// CreateVolume creates a Docker volume for a cluster
// if the volume exists performs no action
// returns the volume name and an error if unsuccessful
func (d *DockerTasks) CreateVolume(name string) (string, error) {
	vn := utils.FQDNVolumeName(name)

	// By default Docker will wildcard searches, use regex to return the absolute
	args := volume.ListOptions{Filters: filters.NewArgs()}
	args.Filters.Add("name", vn)

	ops, err := d.c.VolumeList(context.Background(), args)
	if err != nil {
		return "", xerrors.Errorf("unable to list docker volume '%s': %w", vn, err)
	}

	if len(ops.Volumes) > 0 {
		d.l.Debug("Volume exists", "ref", name, "name", vn)
		return vn, nil
	}

	d.l.Debug("Create Volume", "ref", name, "name", vn)

	volumeCreateOptions := volume.CreateOptions{
		Name:       vn,
		Driver:     "local", //TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	vol, err := d.c.VolumeCreate(context.Background(), volumeCreateOptions)
	if err != nil {
		return "", xerrors.Errorf("failed to create volume '%s': %w", vn, err)
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
		return xerrors.Errorf("unable to copy '%s' from container '%s': %w", src, id, err)
	}
	defer reader.Close()

	// untar the file to a temporary folder
	dir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory, %s", err)
	}

	// clean up the temp dir
	defer os.RemoveAll(dir)

	// make sure the destination does not exist
	if _, err := os.Stat(dst); err == nil {
		err = os.RemoveAll(dst)
		if err != nil {
			return fmt.Errorf("unable to remove destination file or directory: %s", err)
		}
	}

	err = d.tg.Uncompress(reader, false, dir)
	if err != nil {
		return fmt.Errorf("unable to extract tar file: %s", err)
	}

	// copy the source temp to the destination

	// the source file or folder name is the last part of the src
	filename := path.Base(src)
	tmpPath := path.Join(dir, filename)

	// if tmpPath is a directory copy the directory
	i, _ := os.Stat(tmpPath)
	if i.IsDir() {
		err = copyDir(tmpPath, dst)
		if err != nil {
			return fmt.Errorf("unable to copy temporary files to destination %s", err)
		}

		return nil
	}

	// else just copy the file
	_, err = copy(tmpPath, dst)
	if err != nil {
		return fmt.Errorf("unable to copy temporary files to destination %s", err)
	}

	return nil
}

var importMutex = sync.Mutex{}

// CopyLocalDockerImagesToVolume writes multiple Docker images to a Docker container as a compressed archive
// returns the filename of the archive and an error if one occurred
func (d *DockerTasks) CopyLocalDockerImagesToVolume(images []string, volume string, force bool) ([]string, error) {
	d.l.Debug("Writing docker images to volume", "images", images, "volume", volume)

	// make sure this operation runs sequentially as we do not want to update the same volume at the same time
	// for now it should be ok to block globally
	importMutex.Lock()
	defer importMutex.Unlock()

	savedImages := []string{}

	// first check that the images are in the local cache
	for n, i := range images {
		// first check the short tag like envoy-proxy/envoy:latest
		args := filters.NewArgs()
		args.Add("reference", i)

		sum, err := d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
		if err != nil {
			return nil, xerrors.Errorf("unable to list images in local Docker cache: %w", err)
		}

		// we have image
		if len(sum) > 0 {
			continue
		}

		// check the canonical name like docker.io/library/envoy-proxy/envoy:latest as this might be a podman server
		in := makeImageCanonical(i)

		args = filters.NewArgs()
		args.Add("reference", in)

		sum, err = d.c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
		if err != nil {
			return nil, xerrors.Errorf("unable to list images in local Docker cache: %w", err)
		}

		if len(sum) > 0 {
			// update the image name in the collection to the canonical name
			images[n] = in
			continue
		}

		return nil, fmt.Errorf("unable to find image '%s' in the local Docker cache, please pull the image before attempting to copy to a volume", i)
	}

	for _, i := range images {
		compressedImageName := base64.StdEncoding.EncodeToString([]byte(i))

		d.l.Debug("Copying image to container", "image", i)
		imageFile, err := d.saveImageToTempFile(i, compressedImageName)
		if err != nil {
			return nil, err
		}

		// clean up
		defer os.Remove(imageFile)
		savedImages = append(savedImages, imageFile)
	}

	// copy the images to a volume
	return d.CopyFilesToVolume(volume, savedImages, "/images", force)
}

// CopyFileToVolume copies a file to a Docker volume
// returns the names of the stored files
func (d *DockerTasks) CopyFilesToVolume(volumeID string, filenames []string, path string, force bool) ([]string, error) {
	// make sure we have the alpine image needed to copy
	err := d.PullImage(dtypes.Image{Name: "alpine:latest"}, false)
	if err != nil {
		return nil, xerrors.Errorf("unable pull 'alpine:latest' needed to copy files to volume: %w", err)
	}

	// create a dummy container to import to volume
	name := fmt.Sprintf("%d", time.Now().UnixNano())
	name = name[len(name)-8:]
	cc := &dtypes.Container{}
	cc.Name = fmt.Sprintf("%s-import", name)

	cc.Image = &dtypes.Image{Name: "alpine:latest"}
	cc.Volumes = []dtypes.Volume{
		dtypes.Volume{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
	}
	cc.Command = []string{"tail", "-f", "/dev/null"}

	tmpID, err := d.CreateContainer(cc)
	if err != nil {
		return nil, xerrors.Errorf("unable to create dummy container for importing files: %w", err)
	}
	defer d.RemoveContainer(tmpID, true)

	// wait for container to start

	successCount := 0
	failCount := 0
	var startError error
	for {
		i, err := d.c.ContainerInspect(context.Background(), tmpID)
		if err != nil {
			startError = err
			failCount++
			time.Sleep(1 * time.Second)
			continue
		}

		if i.State.Running {
			successCount++

			// wait for 2 positive checks
			if successCount == 2 {
				break
			}

			time.Sleep(1 * time.Second)
			continue
		}

		// state not running increment fail count
		failCount++

		// failed waiting for start
		if failCount == 5 {
			d.l.Error("Timeout waiting for container to start", "ref", tmpID, "error", err)
			startError = xerrors.Errorf("timeout waiting for container to start: %w", startError)

			return nil, startError
		}

		time.Sleep(1 * time.Second)
	}

	// create the directory paths ensure unix paths for containers
	destPath := filepath.ToSlash(filepath.Join("/cache", path))
	_, err = d.ExecuteCommand(tmpID, []string{"mkdir", "-p", destPath}, nil, "/", "", "", 300, nil)
	if err != nil {
		return nil, xerrors.Errorf("unable to create destination path '%s' in volume: %w", destPath, err)
	}

	// add each file individually
	imported := []string{}
	for _, f := range filenames {
		// get the filename part
		name := filepath.Base(f)
		destFile := filepath.Join(destPath, name)

		// check if the image exists if we are not doing a forced update
		if !d.force && !force {
			_, err := d.ExecuteCommand(tmpID, []string{"find", destFile}, nil, "/", "", "", 300, nil)
			if err == nil {
				// we have the image already
				d.l.Debug("File already cached", "name", name, "path", path)
				imported = append(imported, destFile)
				continue
			}
		}

		err = d.CopyFileToContainer(cc.Name, f, destPath)
		if err != nil {
			return nil, xerrors.Errorf("unable to copy file %s to container: %w", f, err)
		}

		imported = append(imported, destFile)
	}

	return imported, nil
}

// CreateFileInContainer creates a file with the given contents and name in the container containerID and
// stores it in the container at the directory path.
func (d *DockerTasks) CreateFileInContainer(containerID, contents, filename, path string) error {
	tmpFile := filepath.Join(os.TempDir(), filename)

	err := os.WriteFile(tmpFile, []byte(contents), 0755)
	if err != nil {
		return xerrors.Errorf("unable to write contents to temporary file: %w", err)
	}

	defer func() {
		os.Remove(tmpFile)
	}()

	err = d.CopyFileToContainer(containerID, tmpFile, path)
	if err != nil {
		return err
	}

	return nil
}

// CopyFileToContainer copies the file at path filename to the container containerID and
// stores it in the container at the directory path.
func (d *DockerTasks) CopyFileToContainer(containerID, filename, path string) error {
	f, err := os.Open(filename)
	if err != nil {
		return xerrors.Errorf("unable to open file '%s': %w", filename, err)
	}
	defer f.Close()

	// create temp file for a tar which will be used to package the file
	// CopyToContainer expects a tar which has individual file entries
	// if we write the original file the output will not be a single file
	// but the contents of the tar. To bypass this we need to add the output from
	// save image to a tar
	tmpTarFile, err := os.CreateTemp("", "")
	if err != nil {
		return xerrors.Errorf("unable to create temporary file for tar archive: %w", err)
	}

	defer func() {
		tmpTarFile.Close()
		os.Remove(tmpTarFile.Name())
	}()

	ta := tar.NewWriter(tmpTarFile)

	fi, _ := f.Stat()

	hdr, err := tar.FileInfoHeader(fi, fi.Name())
	if err != nil {
		return xerrors.Errorf("unable to create file info header for tar: %w", err)
	}

	// write the header to the tar file, this has to happen before the file
	err = ta.WriteHeader(hdr)
	if err != nil {
		return xerrors.Errorf("unable to write tar header: %w", err)
	}

	io.Copy(ta, f)
	if err != nil {
		return xerrors.Errorf("unable to copy image to tar file: %w", err)
	}

	ta.Close()

	// reset the file seek so we can copy to the container
	tmpTarFile.Seek(0, 0)

	err = d.c.CopyToContainer(context.Background(), containerID, path, tmpTarFile, types.CopyToContainerOptions{})
	if err != nil {
		return xerrors.Errorf("unable to copy file to container: %w", err)
	}

	return nil
}

// ExecuteCommand allows the execution of commands in a running docker container
// id is the id of the container to execute the command in
// command is a slice of strings to execute
// writer [optional] will be used to write any output from the command execution.
func (d *DockerTasks) ExecuteCommand(id string, command []string, env []string, workingDir string, user, group string, timeout int, writer io.Writer) (int, error) {
	// set the user details
	if user != "" && group != "" {
		user = fmt.Sprintf("%s:%s", user, group)
	}

	execid, err := d.c.ContainerExecCreate(context.Background(), id, types.ExecConfig{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		Env:          env,
		WorkingDir:   workingDir,
		User:         user,
	})

	if err != nil {
		return defaultExitCode, xerrors.Errorf("unable to create container exec: %w", err)
	}

	// get logs from an attach
	stream, err := d.c.ContainerExecAttach(context.Background(), execid.ID, types.ExecStartCheck{})
	if err != nil {
		return defaultExitCode, xerrors.Errorf("unable to attach logging to exec process: %w", err)
	}

	defer stream.Close()

	streamContext, cancelStream := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelStream()

	// if we have a writer stream the logs from the container to the writer
	if writer != nil {

		ttyOut := streams.NewOut(writer)
		ttyErr := streams.NewOut(writer)

		errCh := make(chan error, 1)

		go func() {
			defer close(errCh)
			errCh <- func() error {
				streamer := streams.NewHijackedStreamer(nil, ttyOut, nil, ttyOut, ttyErr, stream, false, "")

				return streamer.Stream(streamContext)
			}()
		}()

		if err := <-errCh; err != nil {
			d.l.Error("unable to hijack exec stream: %s", err)
			return defaultExitCode, err
		}
	}

	// loop until the container finishes execution
	for {
		i, err := d.c.ContainerExecInspect(context.Background(), execid.ID)
		if err != nil {
			return defaultExitCode, xerrors.Errorf("unable to determine status of exec process: %w", err)
		}

		if !i.Running {
			if i.ExitCode == 0 {
				return i.ExitCode, nil
			}

			return i.ExitCode, xerrors.Errorf("container exec failed with exit code %d", i.ExitCode)
		}

		time.Sleep(1 * time.Second)
	}
}

// ExecuteScript allows the execution of a script in a running docker container
// id is the id of the container to execute the command in
// contents is the contents of the script to execute
// writer [optional] will be used to write any output from the command execution.
func (d *DockerTasks) ExecuteScript(id string, contents string, env []string, workingDir string, user, group string, timeout int, writer io.Writer) (int, error) {
	// ensure we only have unix line ending in ths script
	contents = strings.Replace(contents, "\r\n", "\n", -1)

	// set the user details
	if user != "" && group != "" {
		user = fmt.Sprintf("%s:%s", user, group)
	}

	command := []string{"sh", "/tmp/script.sh"}

	err := d.CreateFileInContainer(id, contents, "script.sh", "/tmp")
	if err != nil {
		return defaultExitCode, xerrors.Errorf("unable to create script in container: %w", err)
	}

	exitCode, err := d.ExecuteCommand(id, command, env, workingDir, user, group, timeout, writer)
	if err != nil {
		return exitCode, err
	}

	return exitCode, nil
}

// TODO: this is all exploratory, works but needs a major tidy

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

	streamContext, streamCancel := context.WithCancel(context.Background())

	go func() {
		defer close(errCh)
		errCh <- func() error {

			streamer := streams.NewHijackedStreamer(ttyIn, ttyOut, ttyIn, ttyOut, ttyErr, resp, true, "")

			return streamer.Stream(streamContext)
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
			streamCancel()
			return xerrors.Errorf("unable to determine status of exec process: %w", err)
		}

		if !i.Running {
			if i.ExitCode == 0 {
				streamCancel()
				return nil
			}

			streamCancel()
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

func (d *DockerTasks) AttachNetwork(net, containerID string, aliases []string, ipAddress string) error {
	d.l.Debug("Attaching container to network", "ref", containerID, "network", net)
	es := &network.EndpointSettings{NetworkID: net}

	// if we have network aliases defined, add them to the network connection
	if len(aliases) > 0 {
		es.Aliases = aliases
	}

	// are we binding to a specific ip
	if ipAddress != "" {
		d.l.Debug("Assigning static ip address", "ref", containerID, "network", net, "ip_address", ipAddress)
		es.IPAMConfig = &network.EndpointIPAMConfig{IPv4Address: ipAddress}
	}

	return d.c.NetworkConnect(context.Background(), net, containerID, es)
}

// ListNetworks lists the networks a container is attached to
func (d *DockerTasks) ListNetworks(id string) []dtypes.NetworkAttachment {
	nets, _ := d.c.NetworkList(context.Background(), types.NetworkListOptions{})

	attachments := []dtypes.NetworkAttachment{}
	for _, n := range nets {
		// get the network destails
		n, err := d.c.NetworkInspect(context.Background(), n.ID, types.NetworkInspectOptions{})

		if err == nil {
			// loop the network and find an address
			for k, c := range n.Containers {
				if k == id {
					att := dtypes.NetworkAttachment{}

					att.ID = n.Labels["id"]
					att.Name = n.Name
					att.IPAddress = c.IPv4Address

					attachments = append(attachments, att)
				}
			}
		}
	}

	return attachments
}

// DetachNetwork detaches a container from a network
// TODO: Docker returns success before removing a container
// tasks which depend on the network being removed may fail in the future
// we need to check it has been removed before returning
func (d *DockerTasks) DetachNetwork(network, containerID string) error {
	network = strings.Replace(network, "network.", "", -1)
	err := d.c.NetworkDisconnect(context.Background(), network, containerID, true)

	// Hacky hack for now
	//time.Sleep(1000 * time.Millisecond)

	return err
}

// FindNetwork returns a network using the unique resource id
func (d *DockerTasks) FindNetwork(id string) (dtypes.NetworkAttachment, error) {
	nets, err := d.c.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		return dtypes.NetworkAttachment{}, err
	}

	for _, n := range nets {
		if n.Labels["id"] == id {
			return dtypes.NetworkAttachment{
				ID:     n.ID,
				Name:   n.Name,
				Subnet: n.IPAM.Config[0].Subnet,
			}, nil
		}
	}

	return dtypes.NetworkAttachment{}, fmt.Errorf("a network with the label id:%s, was not found", id)
}

// publishedPorts defines a Docker published port
type publishedPorts struct {
	ExposedPorts map[nat.Port]struct{}
	PortBindings map[nat.Port][]nat.PortBinding
}

// createPublishedPorts converts a list of config.Port to Docker publishedPorts type
func createPublishedPorts(ps []dtypes.Port) publishedPorts {
	pp := publishedPorts{
		ExposedPorts: make(map[nat.Port]struct{}, 0),
		PortBindings: make(map[nat.Port][]nat.PortBinding, 0),
	}

	for _, p := range ps {
		if p.Protocol == "" {
			p.Protocol = "tcp"
		}

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

func createPublishedPortRanges(ps []dtypes.PortRange) (publishedPorts, error) {
	var portRangeError = xerrors.Errorf("invalid port range, range should be written start-end, e.g 80-82")

	pp := publishedPorts{
		ExposedPorts: make(map[nat.Port]struct{}, 0),
		PortBindings: make(map[nat.Port][]nat.PortBinding, 0),
	}

	for _, p := range ps {
		// split the range
		parts := strings.Split(p.Range, "-")
		if len(parts) != 2 {
			return pp, portRangeError
		}

		// ensure the start is less than the end
		start, serr := strconv.Atoi(parts[0])
		end, eerr := strconv.Atoi(parts[1])

		if serr != nil || eerr != nil {
			return pp, portRangeError
		}

		if start > end {
			return pp, portRangeError
		}

		// range is ok, generate ports
		for i := start; i < end+1; i++ {
			port := strconv.Itoa(i)
			dp, _ := nat.NewPort(p.Protocol, port)
			pp.ExposedPorts[dp] = struct{}{}

			if p.EnableHost {
				pb := []nat.PortBinding{
					nat.PortBinding{
						HostIP:   "0.0.0.0",
						HostPort: port,
					},
				}

				pp.PortBindings[dp] = pb
			}
		}
	}

	return pp, nil
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

// saveImageToTempFile saves a Docker image to a temporary tar file
// it is the responsibility of the caller to remove the temporary file
func (d *DockerTasks) saveImageToTempFile(image, filename string) (string, error) {
	// save the image to a local temp file
	ir, err := d.c.ImageSave(context.Background(), []string{image})
	if err != nil {
		return "", xerrors.Errorf("unable to save images: %w", err)
	}
	defer ir.Close()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", xerrors.Errorf("unable to create temporary file: %w", err)
	}

	// create a temp file to hold the tar
	tmpFileName := path.Join(tmpDir, filename)
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return "", xerrors.Errorf("unable to create temporary file: %w", err)
	}

	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, ir)
	if err != nil {
		return "", xerrors.Errorf("unable to copy image to temp file: %w", err)
	}

	return tmpFileName, nil
}

func copyDir(src string, dest string) error {

	if dest == src {
		return fmt.Errorf("cannot copy a folder into the folder itself!")
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}

	file, err := f.Stat()
	if err != nil {
		return err
	}
	if !file.IsDir() {
		return fmt.Errorf("Source " + file.Name() + " is not a directory!")
	}

	err = os.Mkdir(dest, 0755)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {

		if f.IsDir() {

			err = copyDir(src+"/"+f.Name(), dest+"/"+f.Name())
			if err != nil {
				return err
			}

		}

		if !f.IsDir() {

			content, err := ioutil.ReadFile(src + "/" + f.Name())
			if err != nil {
				return err

			}

			err = ioutil.WriteFile(dest+"/"+f.Name(), content, 0755)
			if err != nil {
				return err

			}

		}

	}

	return nil
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
