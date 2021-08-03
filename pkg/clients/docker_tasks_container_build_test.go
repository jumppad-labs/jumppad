package clients

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func testBuildMockSetup() *mocks.MockDocker {
	// we need to add the stream index (stdout) as the first byte for the hijacker
	writerOutput := []byte("log output")
	writerOutput = append([]byte{1}, writerOutput...)

	mk := &mocks.MockDocker{}
	mk.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]types.ImageSummary{{ID: "abc"}}, nil)
	mk.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(
		types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

	return mk
}

func TestBuildListsImagesAndErrorWhenError(t *testing.T) {
	md := testBuildMockSetup()
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context"}

	dt := NewDockerTasks(md, nil, &TarGz{}, hclog.NewNullLogger())

	_, err := dt.BuildContainer(cc, false)

	assert.Error(t, err)
}
func TestBuildListsImagesAndDoesNotBuildWhenExists(t *testing.T) {
	md := testBuildMockSetup()
	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context"}

	dt := NewDockerTasks(md, nil, &TarGz{}, hclog.NewNullLogger())

	in, err := dt.BuildContainer(cc, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)
}

func TestBuildListsImagesAndBuildsWhenNotExists(t *testing.T) {
	md := testBuildMockSetup()
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	cc := config.NewContainer("test")
	cc.Build = &config.Build{Context: "./context"}

	dt := NewDockerTasks(md, nil, &TarGz{}, hclog.NewNullLogger())

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
	cc.Build = &config.Build{Context: "./context", File: "./Dockerfile-test"}

	dt := NewDockerTasks(md, nil, &TarGz{}, hclog.NewNullLogger())

	in, err := dt.BuildContainer(cc, false)

	assert.NoError(t, err)
	assert.Equal(t, "shipyard.run/localcache/test:latest", in)

	params := getCalls(&md.Mock, "ImageBuild")[0].Arguments[2].(types.ImageBuildOptions)
	assert.Equal(t, "./Dockerfile-test", params.Dockerfile)
}
