package providers

import (
	"testing"

	"github.com/docker/docker/api/types/container"
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

	/* assert correct
	params := md.Calls[0].Arguments

	// check config
	c := params[1].(*container.Config)
	assert.Equal(t, "arse", c.Hostname)

	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
	*/
}
