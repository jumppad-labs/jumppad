package clients

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	clients "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func testBuildMockSetup() *mocks.MockDocker {
	// we need to add the stream index (stdout) as the first byte for the hijacker
	writerOutput := []byte("log output")
	writerOutput = append([]byte{1}, writerOutput...)

	mk := &mocks.MockDocker{}
	mk.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	mk.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]types.ImageSummary{{ID: "abc"}}, nil)
	mk.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(
		types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

	mk.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	return mk
}

func TestBuildListsImagesAndErrorWhenError(t *testing.T) {
	md := testBuildMockSetup()
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context", Tag: "latest"}

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	_, err := dt.BuildContainer(cc, false)

	assert.Error(t, err)
}
func TestBuildListsImagesAndDoesNotBuildWhenExists(t *testing.T) {
	md := testBuildMockSetup()
	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context", Tag: "latest"}

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	in, err := dt.BuildContainer(cc, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)
}

func TestBuildListsImagesAndBuildsWhenNotExists(t *testing.T) {
	md := testBuildMockSetup()
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context", Tag: "latest"}

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	in, err := dt.BuildContainer(cc, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)

	params := getCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile", params.Dockerfile)
}

func TestBuildListsImagesAndBuildsWhenNotExistsCustomDockerfile(t *testing.T) {
	md := testBuildMockSetup()
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context", File: "./Dockerfile-test", Tag: "latest"}

	dt := NewDockerTasks(md, nil, &TarGz{}, clients.NewTestLogger(t))

	in, err := dt.BuildContainer(cc, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)

	params := getCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile-test", params.Dockerfile)
}
