package container

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	imocks "github.com/jumppad-labs/jumppad/pkg/clients/images/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testCopyLocalImages = []string{"consul:1.6.1"}
var testCopyLocalVolume = "images"

// Create happy path mocks
func testSetupCopyLocal(t *testing.T) (*DockerTasks, *mocks.Docker) {
	mk := &mocks.Docker{}
	mk.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	mk.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)
	mk.On("ContainerInspect", mock.Anything, mock.Anything).Return(
		container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{State: &container.State{Running: true}},
		},
		nil)

	mk.On("ImageSave", mock.Anything, mock.Anything).Return(
		io.NopCloser(bytes.NewBufferString("test")),
		nil,
	)

	mk.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.CreateResponse{ID: "myid"}, nil)

	mk.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mk.On(
		"CopyToContainer",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(nil)

	mk.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// always return a local image
	mk.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]image.Summary{{}}, nil)

	mk.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).
		Return(io.NopCloser(strings.NewReader("hello world")), nil)

	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).
		Return(container.ExecCreateResponse{ID: "abc"}, nil)

	mk.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(
		types.HijackedResponse{
			Conn: &net.TCPConn{},
			Reader: bufio.NewReader(
				bytes.NewReader([]byte("log output")),
			),
		},
		nil,
	)

	mk.On("ContainerExecStart", mock.Anything, "abc", mock.Anything).Return(nil)

	mk.On("ContainerExecInspect", mock.Anything, "abc", mock.Anything).
		Return(container.ExecInspect{Running: false, ExitCode: 0}, nil)

	mk.On("VolumeList", mock.Anything, mock.Anything).
		Return(volume.ListResponse{Volumes: []*volume.Volume{{}}})

	mk.On("NetworkList", mock.Anything, mock.Anything).
		Return([]network.Inspect{{
			Name: "jumppad",
			ID:   "resource.network.jumppad",
			Labels: map[string]string{
				"id": "resource.network.jumppad",
			},
			IPAM: network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{
					{
						Subnet: "10.0.10.0/24",
					},
				},
			},
			EnableIPv6: false,
		}}, nil)

	mk.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mic := &imocks.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	dt, _ := NewDockerTasks(mk, mic, &tar.TarGz{}, logger.NewTestLogger(t))
	dt.defaultWait = 1 * time.Millisecond

	return dt, mk
}

func TestCopyToVolumeDoesNothingWhenCached(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)

	// when force is set ContainerStop is called
	//mk.On("ContainerStop",mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)

	args := container.ExecOptions{
		Cmd: []string{
			"find",
			"/cache/images/" +
				base64.StdEncoding.EncodeToString([]byte(testCopyLocalImages[0])),
		},
		WorkingDir:   "/",
		AttachStdout: true,
		AttachStderr: true,
	}

	mk.AssertCalled(t, "ContainerExecCreate", mock.Anything, "myid", args)
	mk.AssertNotCalled(t, "ImageSave")
}

func TestCopyToVolumeDoesNotChecksVolumeCacheWhenGlobalForce(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)

	// should have been called once to create the directory
	mk.AssertNumberOfCalls(t, "ContainerExecCreate", 1)
}

func TestCopyToVolumeDoesNotChecksVolumeCacheWhenLocalForce(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(false) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, true)
	assert.NoError(t, err)

	// should have been called once to create the directory
	mk.AssertNumberOfCalls(t, "ContainerExecCreate", 1)
}

func TestCopyToVolumeSavesImages(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)
	mk.AssertCalled(t, "ImageSave", mock.Anything, testCopyLocalImages)
}

func TestCopyToVolumeSavesImageFailReturnsError(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	testutils.RemoveOn(&mk.Mock, "ImageSave")
	mk.On("ImageSave", mock.Anything, mock.Anything).Return(
		nil,
		fmt.Errorf("blah"),
	)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.Error(t, err)
}

func TestCopyToVolumeCreatesTempContainer(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)

	// ensure it mounts the volume
	params := testutils.GetCalls(&mk.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)
	cfg := params[1].(*container.Config)

	// test name and command
	assert.Contains(t, cfg.Hostname, "import")
	assert.Equal(t, "tail", cfg.Cmd[0])

	// test mounts volume
	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, "/cache", hc.Mounts[0].Target)
	assert.Equal(t, "images", hc.Mounts[0].Source)
}

func TestCopyToVolumeChecksTempContainerStart(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "ContainerInspect", 2)
}

func TestCopyToVolumeReturnsErrorOnFailedContainerStart(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)

	testutils.RemoveOn(&mk.Mock, "ContainerInspect")
	mk.On("ContainerInspect", mock.Anything, mock.Anything).Return(
		types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{State: &types.ContainerState{Running: false}},
		},
		nil)

	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.Error(t, err)

	mk.AssertNumberOfCalls(t, "ContainerInspect", 5)
}

func TestCopyToVolumeTempContainerFailsReturnError(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	testutils.RemoveOn(&mk.Mock, "ContainerCreate")
	mk.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.CreateResponse{}, fmt.Errorf("boom"))

	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.Error(t, err)
}

func TestCopyToVolumePullsImportImage(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)
	mk.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical("alpine:latest"), mock.Anything)
}

func TestCopyToVolumeCreatesDestinationDirectory(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecCreate", mock.Anything, "myid", mock.Anything)

	params := testutils.GetCalls(&mk.Mock, "ContainerExecCreate")[0].Arguments[2].(container.ExecOptions)
	assert.Equal(t, []string{"mkdir", "-p", "/cache/images"}, params.Cmd)
}

func TestCopyToVolumeCopiesArchive(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)
	mk.AssertCalled(t, "CopyToContainer", mock.Anything, mock.Anything, "/cache/images", mock.Anything, mock.Anything)
}

func TestCopyToVolumeCopiesArchiveFailReturnsError(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	testutils.RemoveOn(&mk.Mock, "CopyToContainer")
	mk.On("CopyToContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.Error(t, err)
}

func TestCopyToVolumeRemovesTempContainer(t *testing.T) {
	dt, mk := testSetupCopyLocal(t)
	dt.SetForce(true) // set force pull to avoid execute command block

	_, err := dt.CopyLocalDockerImagesToVolume(testCopyLocalImages, testCopyLocalVolume, false)
	assert.NoError(t, err)
	mk.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}
