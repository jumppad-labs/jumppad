package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"golang.org/x/xerrors"
)

var (
	ErrorClusterInvalidName = errors.New("invalid cluster name")
)

const k3sBaseImage = "rancher/k3s"

func (c *Cluster) createK3s() error {
	// check the cluster name is valid
	if err := validateClusterName(c.config.Name); err != nil {
		return err
	}

	// check the cluster does not already exist
	id, err := c.Lookup()
	if err == nil || id != "" {
		return ErrorClusterExists
	}

	// set the image
	image := fmt.Sprintf("%s:%s", k3sBaseImage, c.config.Version)

	// create the server
	dc := &container.Config{
		Hostname: c.config.Name,
		Image:    image,
	}

	/*
		// Create volume mounts
		mounts := []mount.Mount{}
		for _, vc := range c.config.Volumes {
			sourcePath, err := filepath.Abs(vc.Source)
			if err != nil {
				return err
			}

			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: sourcePath,
				Target: vc.Destination,
			})
		}

		hc := &container.HostConfig{
			Mounts: mounts,
		}
	*/

	hc := &container.HostConfig{}
	nc := &network.NetworkingConfig{}

	cont, err := c.client.ContainerCreate(
		context.Background(),
		dc,
		hc,
		nc,
		c.config.Name,
	)

	c.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})

	return nil
}

const clusterNameMaxSize int = 35

func validateClusterName(name string) error {
	if err := validateHostname(name); err != nil {
		return err
	}

	if len(name) > clusterNameMaxSize {
		return xerrors.Errorf("cluster name is too long (%d > %d): %w", len(name), clusterNameMaxSize, ErrorClusterInvalidName)
	}

	return nil
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func validateHostname(name string) error {
	if len(name) == 0 {
		return xerrors.Errorf("no name provided %w", ErrorClusterInvalidName)
	}

	if name[0] == '-' || name[len(name)-1] == '-' {
		return xerrors.Errorf("hostname [%s] must not start or end with - (dash): %w", name, ErrorClusterInvalidName)
	}

	for _, c := range name {
		switch {
		case '0' <= c && c <= '9':
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '-':
			break
		default:
			return xerrors.Errorf("hostname [%s] contains characters other than 'Aa-Zz', '0-9' or '-': %w", ErrorClusterInvalidName)

		}
	}

	return nil
}
