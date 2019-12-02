package providers

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/shipyard-run/cli/config"
	"golang.org/x/xerrors"
)

var (
	ErrorClusterInvalidName = errors.New("invalid cluster name")
)

// https://github.com/rancher/k3d/blob/master/cli/commands.go

const k3sBaseImage = "rancher/k3s"

func (c *Cluster) createK3s() error {
	// check the cluster name is valid
	if err := validateClusterName(c.config.Name); err != nil {
		return err
	}

	// check the cluster does not already exist
	id, err := c.Lookup()
	if err == nil && id != "" {
		return ErrorClusterExists
	}

	// set the image
	image := fmt.Sprintf("%s:%s", k3sBaseImage, c.config.Version)

	// create the server
	// since the server is just a container create the container config and provider
	cc := &config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.Image = image
	cc.NetworkRef = c.config.NetworkRef

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = []config.KV{
		config.KV{Key: "K3S_KUBECONFIG_OUTPUT", Value: "/output/kubeconfig.yaml"},
		config.KV{Key: "K3S_KUBECONFIG_OUTPUT", Value: "mysupersecret"}, // This should be random
	}

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000
	args := []string{"server", fmt.Sprintf("--api-port=%d", apiPort)}

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    apiPort,
			Host:     apiPort,
			Protocol: "tcp",
		},
	}

	// disable the installation of traefik
	args = append(args, "--server-arg=\"--no-deploy=traefik\"")

	cc.Command = args

	cp := NewContainer(cc, c.client)
	return cp.Create()
}

func (c *Cluster) destroyK3s() error {
	cc := &config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.NetworkRef = c.config.NetworkRef

	cp := NewContainer(cc, c.client)
	return cp.Destroy()
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
