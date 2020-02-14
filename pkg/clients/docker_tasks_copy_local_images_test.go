package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testCopyLocalImages = []string{"consul:1.6.1"}
var testCopyLocalVolume = "images"

// Create happy path mocks
func testCreateCopyLocalMocks() *mocks.MockDocker {
	mk := &mocks.MockDocker{}
	mk.On("ImageSave", mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(bytes.NewBufferString("test")),
		nil,
	)
	mk.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	mk.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.On("CopyToContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return mk
}

func TestCopyLocalSavesImages(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.NoError(t, err)
	mk.AssertCalled(t, "ImageSave", mock.Anything, testCopyLocalImages)
}

func TestCopyLocalSavesImageFailReturnsError(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	removeOn(&mk.Mock, "ImageSave")
	mk.On("ImageSave", mock.Anything, mock.Anything).Return(
		nil,
		fmt.Errorf("blah"),
	)
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.Error(t, err)
}

func TestCopyLocalCreatesTempContainer(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.NoError(t, err)

	// ensure it mounts the volume
	params := getCalls(&mk.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)
	cfg := params[1].(*container.Config)

	// test name and command
	assert.Equal(t, "temp-import", cfg.Hostname)
	assert.Equal(t, "tail", cfg.Cmd[0])

	// test mounts volume
	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, testCopyLocalVolume, hc.Mounts[0].Source)
	assert.Equal(t, "/images", hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeVolume, hc.Mounts[0].Type)
}

func TestCopyLocalTempContainerFailsReturnError(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	removeOn(&mk.Mock, "ContainerCreate")
	mk.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, fmt.Errorf("boom"))
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.Error(t, err)
}

func TestCopyLocalCopiesArchive(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.NoError(t, err)
	mk.AssertCalled(t, "CopyToContainer", mock.Anything, "temp-import.container.shipyard", "/images", mock.Anything, mock.Anything)
}

func TestCopyLocalCopiesArchiveFailReturnsError(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	removeOn(&mk.Mock, "CopyToContainer")
	mk.On("CopyToContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.Error(t, err)
}

func TestCopyLocalRemovesTempContainer(t *testing.T) {
	mk := testCreateCopyLocalMocks()
	dt := NewDockerTasks(mk, hclog.NewNullLogger())

	_, err := dt.CopyLocalDockerImageToVolume(testCopyLocalImages, testCopyLocalVolume)
	assert.NoError(t, err)
	mk.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}
