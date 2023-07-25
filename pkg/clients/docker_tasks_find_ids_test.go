package clients

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	clients "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFindContainerIDsReturnsID(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return(
		[]types.Container{
			types.Container{ID: "abc"},
			types.Container{ID: "123"},
		},
		nil,
	)

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	ids, err := dt.FindContainerIDs("test", "cloud")
	assert.NoError(t, err)

	// assert that the docker api call was made
	md.AssertNumberOfCalls(t, "ContainerList", 1)

	// ensure that the FQDN was passed as an argument
	args := getCalls(&md.Mock, "ContainerList")[0].Arguments[1].(types.ContainerListOptions)
	assert.Equal(t, "^test.cloud.shipyard.run$", args.Filters.Get("name")[0])

	// ensure that the id has been returned
	assert.Len(t, ids, 2)
	assert.Equal(t, "abc", ids[0])
	assert.Equal(t, "123", ids[1])
}

func TestFindContainerIDsReturnsErrorWhenDockerFail(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	_, err := dt.FindContainerIDs("test", "cloud")
	assert.Error(t, err)
}

func TestFindContainerIDsReturnsNilWhenNoIDs(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	ids, err := dt.FindContainerIDs("test", "cloud")
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestFindContainerIDsReturnsNilWhenEmpty(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{}, nil)

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	ids, err := dt.FindContainerIDs("test", "cloud")
	assert.NoError(t, err)
	assert.Nil(t, ids)
}
