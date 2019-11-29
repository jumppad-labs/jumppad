package providers

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	clients "github.com/shipyard-run/cli/clients/mocks"
	"github.com/shipyard-run/cli/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/xerrors"
)

func setupK3sCluster(c *config.Cluster) (*clients.MockDocker, *Cluster) {
	md := &clients.MockDocker{}
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
	c := &config.Cluster{Name: "hostname", Driver: "k3s"}
	md, p := setupK3sCluster(c)
	removeOn(&md.Mock, "ContainerList")
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{types.Container{ID: "123sdsdsd"}}, nil)

	err := p.Create()

	md.AssertCalled(t, "ContainerList", mock.Anything, mock.Anything)

	assert.True(t, xerrors.Is(err, ErrorClusterExists))
}

func TestK3sClusterServerCreatesWithCorrectOptions(t *testing.T) {
	c := &config.Cluster{
		Name:    "hostname",
		Driver:  "k3s",
		Version: "v1.0.0",
	}

	md, p := setupK3sCluster(c)

	err := p.Create()

	assert.NoError(t, err)
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// assert server properties
	params := md.Calls[1].Arguments
	dc := params[1].(*container.Config)

	assert.Equal(t, fmt.Sprintf("%s:%s", k3sBaseImage, c.Version), dc.Image)
	assert.Equal(t, c.Name, dc.Hostname)
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
