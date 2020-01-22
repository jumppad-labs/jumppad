package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"archive/tar"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

// pullImage pulls a Docker image from a remote repo
func pullImage(c clients.Docker, image config.Image, l hclog.Logger) error {
	in := makeImageCanonical(image.Name)

	args := filters.NewArgs()
	args.Add("reference", image.Name)

	// only pull if image is not in current registry
	sum, err := c.ImageList(context.Background(), types.ImageListOptions{Filters: args})
	if err != nil {
		return xerrors.Errorf("unable to list images in local Docker cache: %w", err)
	}

	// if we have images do not pull
	if len(sum) > 0 {
		l.Debug("Image exists in local cache", "image", image.Name)

		return nil
	}

	ipo := types.ImagePullOptions{}

	// if the username and password is not null make an authenticated
	// image pull
	if image.Username != "" && image.Password != "" {
		// credentials are a json string and need to be bas64 encoded
		ipo.RegistryAuth = base64.StdEncoding.EncodeToString(
			[]byte(
				fmt.Sprintf(`{"Username": "%s", "Password": "%s"}`, image.Username, image.Password),
			),
		)
	}

	l.Debug("Pulling image", "image", image.Name)

	out, err := c.ImagePull(context.Background(), in, ipo)
	if err != nil {
		return xerrors.Errorf("Error pulling image: %w", err)
	}

	// write the output to /dev/null
	// TODO this stuff needs to be logged correctly
	io.Copy(ioutil.Discard, out)

	return nil
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

// writeLocalDockerImageToVolume writes a docker image to a Docker volume
// returns the filename and an error if one occured
func writeLocalDockerImageToVolume(c clients.Docker, images []config.Image, volume string, l hclog.Logger) (string, error) {
	l.Debug("Writing docker images to volume", "images", images, "volume", volume)

	// temporary slice of image names for the save command
	ins := []string{}

	// make sure that the given image has been pulled locally before saving
	for _, i := range images {

		err := pullImage(c, i, l)
		if err != nil {
			return "", xerrors.Errorf("unable to pull image %s: %w", i, err)
		}

		ins = append(ins, i.Name)
	}

	// save the image to a local temp file
	ir, err := c.ImageSave(context.Background(), ins)
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
	cc := &config.Container{
		Name:  "tmp.import",
		Image: config.Image{Name: makeImageCanonical("alpine:latest")},
		Volumes: []config.Volume{
			config.Volume{
				Source:      volume,
				Destination: "/images",
				Type:        "volume",
			},
		},
		Command: []string{"tail", "-f", "/dev/null"},
	}

	con := NewContainer(cc, c, l)
	err = con.Create()
	if err != nil {
		return "", xerrors.Errorf("unable to create dummy container for importing images: %w", err)
	}
	defer con.Destroy()

	err = c.CopyToContainer(context.Background(), FQDN(cc.Name, nil), "/images", tmpTarFile, types.CopyToContainerOptions{})
	if err != nil {
		return "", xerrors.Errorf("unable to copy file to container: %w", err)
	}

	return fmt.Sprintf("/images/%s", fi.Name()), nil
}

// execute a command in a container
func execCommand(c clients.Docker, container string, command []string, l hclog.Logger) error {
	id, err := c.ContainerExecCreate(context.Background(), container, types.ExecConfig{
		Cmd:          command,
		WorkingDir:   "/",
		AttachStdout: true,
		AttachStderr: true,
	})

	if err != nil {
		return xerrors.Errorf("unable to create container exec: %w", err)
	}

	// get logs from an attach
	stream, err := c.ContainerExecAttach(context.Background(), id.ID, types.ExecStartCheck{})
	if err != nil {
		return xerrors.Errorf("unable to attach logging to exec process: %w", err)
	}
	defer stream.Close()

	// ensure that the log from the Docker exec command is copied to the default logger
	go func() {
		io.Copy(
			l.StandardWriter(&hclog.StandardLoggerOptions{}),
			stream.Reader,
		)
	}()

	err = c.ContainerExecStart(context.Background(), id.ID, types.ExecStartCheck{})
	if err != nil {
		return xerrors.Errorf("unable to start exec process: %w", err)
	}

	// loop until the container finishes execution
	for {
		i, err := c.ContainerExecInspect(context.Background(), id.ID)
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

	return nil
}
