package container

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	dtypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func testBuildSetup(t *testing.T) (*mocks.Docker, *DockerTasks) {
	// we need to add the stream index (stdout) as the first byte for the hijacker
	writerOutput := []byte("log output")
	writerOutput = append([]byte{1}, writerOutput...)

	mk := &mocks.Docker{}
	mk.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	mk.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]types.ImageSummary{{ID: "abc"}}, nil)
	mk.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(
		types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

	mk.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	dt, _ := NewDockerTasks(mk, nil, &tar.TarGz{}, logger.NewTestLogger(t))

	return mk, dt
}

func TestBuildListsImagesAndErrorWhenError(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	b := &dtypes.Build{Context: "../../../examples/build/src"}

	_, err := dt.BuildContainer(b, false)

	assert.Error(t, err)
}

func TestBuildListsImagesAndDoesNotBuildWhenExists(t *testing.T) {
	_, dt := testBuildSetup(t)
	b := &dtypes.Build{Name: "test", Context: "../../../examples/build/src"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Contains(t, in, "jumppad.dev/localcache/test:")
}

func TestBuildListsImagesAndBuildsWhenNotExists(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	b := &dtypes.Build{Name: "test", Context: "../../../examples/build/src"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Contains(t, in, "jumppad.dev/localcache/test:")

	params := testutils.GetCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile", params.Dockerfile)
}

func TestBuildListsImagesAndBuildsWhenNotExistsCustomDockerfile(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	b := &dtypes.Build{Name: "test", Context: "../../../examples/build/src", DockerFile: "./Docker/Dockerfile"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Contains(t, in, "jumppad.dev/localcache/test:")

	params := testutils.GetCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Docker/Dockerfile", params.Dockerfile)
}
