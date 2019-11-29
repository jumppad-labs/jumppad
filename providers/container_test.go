package providers

import (
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/container"
	// "github.com/docker/docker/api/types/network"
	clients "github.com/shipyard-run/cli/clients/mocks"
	"github.com/shipyard-run/cli/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupContainer(c *config.Container) (*clients.MockDocker, *Container) {
	md := &clients.MockDocker{}
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md, NewContainer(c, md)
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cn := &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
	cc := &config.Container{Name: "testcontainer", Image: "consul:v1.6.1", NetworkRef: cn, Volumes: []config.Volume{config.Volume{Source: "data", Destination: "/data"}}}
	md, p := setupContainer(cc)

	err := p.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	params := md.Calls[0].Arguments

	cfg := params[1].(*container.Config)
	assert.Equal(t, "testcontainer", cfg.Hostname)
	assert.Equal(t, "consul:v1.6.1", cfg.Image)
	// assert.Equal(t, true, cfg.AttachStdin)
	// assert.Equal(t, true, cfg.AttachStdout)
	// assert.Equal(t, true, cfg.AttachStderr)
	// assert.Equal(t, true, cfg.Tty)

	host := params[2].(*container.HostConfig)
	sourcePath, err := filepath.Abs(cc.Volumes[0].Source)
	assert.NoError(t, err)

	destPath := cc.Volumes[0].Destination
	assert.Equal(t, sourcePath, host.Mounts[0].Source)
	assert.Equal(t, destPath, host.Mounts[0].Target)

	// network := params[3].(*network.NetworkingConfig)
	// assert.Equal(t, )

	name := params[4].(string)
	assert.Equal(t, FQDN(cc.Name, cn.Name), name)
}
