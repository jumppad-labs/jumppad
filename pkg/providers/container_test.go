package providers

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupContainer(c *config.Container) (*clients.MockDocker, *Container) {
	md := &clients.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md, NewContainer(c, md)
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cn := &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
	cc := &config.Container{Name: "testcontainer", Image: config.Image{Name: "consul:v1.6.1"}, NetworkRef: cn, Volumes: []config.Volume{config.Volume{Source: "/mnt/data", Destination: "/data"}}}
	md, p := setupContainer(cc)

	err := p.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	host := params[2].(*container.HostConfig)
	network := params[3].(*network.NetworkingConfig)

	cfg := params[1].(*container.Config)
	assert.Equal(t, "testcontainer", cfg.Hostname)
	assert.Equal(t, "consul:v1.6.1", cfg.Image)
	// assert.Equal(t, true, cfg.AttachStdin)
	// assert.Equal(t, true, cfg.AttachStdout)
	// assert.Equal(t, true, cfg.AttachStderr)
	// assert.Equal(t, true, cfg.Tty)

	sourcePath := cc.Volumes[0].Source
	destPath := cc.Volumes[0].Destination
	assert.Equal(t, sourcePath, host.Mounts[0].Source)
	assert.Equal(t, destPath, host.Mounts[0].Target)

	assert.NotNil(t, network.EndpointsConfig[cn.Name])
	assert.Equal(t, cn.Name, network.EndpointsConfig[cn.Name].NetworkID)

	name := params[4].(string)
	assert.Equal(t, FQDN(cc.Name, cn), name)
}
