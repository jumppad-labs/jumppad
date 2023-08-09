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

	dt := NewDockerTasks(mk, nil, &tar.TarGz{}, logger.NewTestLogger(t))

	return mk, dt
}

func TestBuildListsImagesAndErrorWhenError(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	b := &dtypes.Build{Context: "./context"}

	_, err := dt.BuildContainer(b, false)

	assert.Error(t, err)
}
func TestBuildListsImagesAndDoesNotBuildWhenExists(t *testing.T) {
	_, dt := testBuildSetup(t)
	b := &dtypes.Build{Context: "./context"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)
}

func TestBuildListsImagesAndBuildsWhenNotExists(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	b := &dtypes.Build{Context: "./context"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)

	params := testutils.GetCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile", params.Dockerfile)
}

func TestBuildListsImagesAndBuildsWhenNotExistsCustomDockerfile(t *testing.T) {
	md, dt := testBuildSetup(t)
	testutils.RemoveOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	b := &dtypes.Build{Context: "./context", DockerFile: "./Dockerfile-test"}

	in, err := dt.BuildContainer(b, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)

	params := testutils.GetCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile-test", params.Dockerfile)
}
