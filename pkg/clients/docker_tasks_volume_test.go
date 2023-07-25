package clients

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	clients "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateVolumeDoesNothingWhenVolumeExists(t *testing.T) {
	_, _, _, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	removeOn(&md.Mock, "VolumeList")
	f := filters.NewArgs()
	f.Add("name", "test.volume.shipyard.run")
	md.On("VolumeList", mock.Anything, f).Return(volume.VolumeListOKBody{Volumes: []*types.Volume{&types.Volume{}}}, nil)

	_, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeReturnsErrorWhenVolumeListError(t *testing.T) {
	_, _, _, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	removeOn(&md.Mock, "VolumeList")
	f := filters.NewArgs()
	f.Add("name", "test.volume.shipyard.run")
	md.On("VolumeList", mock.Anything, f).Return(nil, fmt.Errorf("Boom"))

	_, err := p.CreateVolume("test")
	assert.Error(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeCreatesSuccesfully(t *testing.T) {
	_, _, _, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	id, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeCreate", mock.Anything, mock.Anything)
	assert.Equal(t, "test_volume", id)
}

func TestRemoveVolumeRemotesSuccesfully(t *testing.T) {
	_, _, _, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	err := p.RemoveVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeRemove", mock.Anything, "test.volume.shipyard.run", true)
}
