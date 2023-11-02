package container

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	dtypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTagImageTagstheImage(t *testing.T) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ImageTag", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	dt, err := NewDockerTasks(md, nil, nil, logger.NewTestLogger(t))
	require.NoError(t, err)

	err = dt.TagImage("abc", "def")
	require.NoError(t, err)

	md.AssertCalled(t, "ImageTag", mock.Anything, "abc", "def")
}

func TestPushPushestheImageToTheRegistryWithoutAuth(t *testing.T) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ImagePush", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(&bytes.Buffer{}), nil)

	dt, err := NewDockerTasks(md, nil, nil, logger.NewTestLogger(t))
	require.NoError(t, err)

	err = dt.PushImage(dtypes.Image{Name: "myimage:latest"})
	require.NoError(t, err)

	md.AssertCalled(t, "ImagePush", mock.Anything, "myimage:latest", mock.Anything)

	// ensure auth is not set
	args := md.Calls[2].Arguments
	auth := args.Get(2).(types.ImagePushOptions).RegistryAuth
	require.Empty(t, auth)
}

func TestPushPushestheImageToTheRegistryWithAuth(t *testing.T) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	md.On("ImagePush", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(&bytes.Buffer{}), nil)

	dt, err := NewDockerTasks(md, nil, nil, logger.NewTestLogger(t))
	require.NoError(t, err)

	err = dt.PushImage(dtypes.Image{Name: "myimage:latest", Username: "user", Password: "pass"})
	require.NoError(t, err)

	md.AssertCalled(t, "ImagePush", mock.Anything, "myimage:latest", mock.Anything)

	// ensure auth is not set
	args := md.Calls[2].Arguments
	auth := args.Get(2).(types.ImagePushOptions).RegistryAuth
	authString, _ := base64.StdEncoding.DecodeString(auth)

	require.Contains(t, string(authString), "user")
	require.Contains(t, string(authString), "pass")
}
