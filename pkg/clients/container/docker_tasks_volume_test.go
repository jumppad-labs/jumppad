package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateVolumeDoesNothingWhenVolumeExists(t *testing.T) {
	_, md, mic := createContainerConfig()

	testutils.RemoveOn(&md.Mock, "VolumeList")
	f := filters.NewArgs()
	f.Add("name", "test.volume.shipyard.run")
	md.On("VolumeList", mock.Anything, f).Return(volume.ListResponse{Volumes: []*volume.Volume{&volume.Volume{}}}, nil)

	p := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))
	_, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeReturnsErrorWhenVolumeListError(t *testing.T) {
	_, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	testutils.RemoveOn(&md.Mock, "VolumeList")
	f := filters.NewArgs()
	f.Add("name", "test.volume.shipyard.run")
	md.On("VolumeList", mock.Anything, f).Return(nil, fmt.Errorf("Boom"))

	_, err := p.CreateVolume("test")
	assert.Error(t, err)

	md.AssertNotCalled(t, "VolumeCreate")
}

func TestCreateVolumeCreatesSuccesfully(t *testing.T) {
	_, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	id, err := p.CreateVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeCreate", mock.Anything, mock.Anything)
	assert.Equal(t, "test_volume", id)
}

func TestRemoveVolumeRemotesSuccesfully(t *testing.T) {
	_, md, mic := createContainerConfig()
	p := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	err := p.RemoveVolume("test")
	assert.NoError(t, err)

	md.AssertCalled(t, "VolumeRemove", mock.Anything, "test.volume.shipyard.run", true)
}
