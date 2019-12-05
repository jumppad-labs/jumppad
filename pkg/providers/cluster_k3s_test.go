package providers

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	clients "github.com/shipyard-run/cli/pkg/clients/mocks"
	"github.com/shipyard-run/cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/xerrors"
)

func setupK3sCluster(c *config.Cluster) (*clients.MockDocker, *Cluster) {
	md := &clients.MockDocker{}
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("container not found"))

	return md, NewCluster(c, md)
}

func TestK3sInvalidClusterNameReturnsError(t *testing.T) {
	c := &config.Cluster{Name: "-hostname.1231", Driver: "k3s"}
	_, p := setupK3sCluster(c)

	err := p.Create()

	assert.True(t, xerrors.Is(err, ErrorClusterInvalidName))
}

func TestK3sReturnsErrorIfClusterExists(t *testing.T) {
	cn := &config.Network{Name: "k3snet"}
	c := &config.Cluster{Name: "hostname", Driver: "k3s", NetworkRef: cn}

	md, p := setupK3sCluster(c)
	removeOn(&md.Mock, "ContainerList")
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{types.Container{ID: "123sdsdsd"}}, nil)

	err := p.Create()

	md.AssertCalled(t, "ContainerList", mock.Anything, mock.Anything)

	assert.True(t, xerrors.Is(err, ErrorClusterExists))
}

func TestK3sClusterServerCreatesWithCorrectOptions(t *testing.T) {
	cn := &config.Network{
		Name: "k3snet",
	}

	c := &config.Cluster{
		Name:       "hostname",
		Driver:     "k3s",
		Version:    "v1.0.0",
		NetworkRef: cn,
	}

	md, p := setupK3sCluster(c)

	err := p.Create()

	assert.NoError(t, err)
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// assert server properties
	params := md.Calls[2].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)
	fqdn := params[4]

	assert.Equal(t, fmt.Sprintf("%s:%s", k3sBaseImage, c.Version), dc.Image)

	// check cluster names
	assert.Equal(t, "server.hostname.k3snet.shipyard", fqdn, "FQDN should be [nodetype].[name].[network].shipyard")
	assert.Equal(t, "server.hostname", dc.Hostname, "Hostname should be [nodetype].[hostname]")

	// check environment variables
	assert.Len(t, dc.Env, 2)
	assert.Equal(t, "K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml", dc.Env[0])
	assert.Contains(t, dc.Env[1], "K3S_KUBECONFIG_OUTPUT=")

	// check the command
	assert.Equal(t, "server", dc.Cmd[0])
	assert.Contains(t, dc.Cmd[1], "--https-listen-port=")
	assert.Equal(t, dc.Cmd[2], "--no-deploy=traefik")

	// check the ports
	apiPort := strings.Split(dc.Cmd[1], "=")
	dockerPort, _ := nat.NewPort("tcp", apiPort[1])
	fmt.Println(dockerPort)

	fmt.Println(hc.PortBindings)
	fmt.Println(dc.ExposedPorts)

	assert.Len(t, dc.ExposedPorts, 1)
	assert.NotNil(t, dc.ExposedPorts[dockerPort])
	assert.Len(t, hc.PortBindings, 1)
	assert.NotNil(t, hc.PortBindings[dockerPort])
}

// removeOn is a utility function for removing Expectations from mock objects
func removeOn(m *mock.Mock, name string) {
	ec := m.ExpectedCalls
	rc := make([]*mock.Call, 0)

	for _, c := range ec {
		if c.Method != name {
			rc = append(rc, c)
		}
	}

	m.ExpectedCalls = rc
}
