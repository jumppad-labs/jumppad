package providers

import (
	"testing"

	"github.com/docker/engine/api/types/container"
	// "github.com/docker/engine/api/types/network"
	clients "github.com/shipyard-run/cli/clients/mocks"
	"github.com/shipyard-run/cli/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupContainer(c *config.Container) (*clients.MockDocker, *Container) {
	md := &clients.MockDocker{}
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)

	return md, NewContainer(c, md)
}

func TestContainerCreatesCorrectly(t *testing.T) {
	md, p := setupContainer(&config.Container{Name: "testcontainer", Image: "consul:v1.6.1"})

	err := p.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	params := md.Calls[0].Arguments

	cfg := params[1].(*container.Config)
	assert.Equal(t, "testcontainer", cfg.Hostname)
	assert.Equal(t, "consul:v1.6.1", cfg.Image)
	// assert.Equal(t, true, cfg.AttachStdin)
	// assert.Equal(t, true, cfg.AttachStdout)
	// assert.Equal(t, true, cfg.AttachStderr)
	// assert.Equal(t, true, cfg.Tty)

	// network := params[3].(*network.NetworkingConfig)
	// assert.Equal(t, )

	name := params[4].(*string)
	assert.Equal(t, "testcontainer", name)
}
