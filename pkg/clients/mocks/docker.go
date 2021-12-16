package mocks

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	volumetypes "github.com/docker/docker/api/types/volume"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
)

type MockDocker struct {
	mock.Mock
}

func (m *MockDocker) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, refStr, options)

	if rc, ok := args.Get(0).(io.ReadCloser); ok {
		return rc, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockDocker) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *specs.Platform,
	containerName string,
) (container.ContainerCreateCreatedBody, error) {

	args := m.Called(ctx, config, hostConfig, networkingConfig, containerName)

	if c, ok := args.Get(0).(container.ContainerCreateCreatedBody); ok {
		return c, args.Error(1)
	}

	return container.ContainerCreateCreatedBody{}, args.Error(1)
}

func (m *MockDocker) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	args := m.Called(ctx, options)

	if cl, ok := args.Get(0).([]types.Container); ok {
		return cl, nil
	}

	return nil, args.Error(1)
}

func (m *MockDocker) ContainerStart(ctx context.Context, ID string, opts types.ContainerStartOptions) error {
	args := m.Called(ctx, ID, opts)

	return args.Error(0)
}

func (m *MockDocker) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	args := m.Called(ctx, containerID, timeout)

	return args.Error(0)
}

func (m *MockDocker) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	args := m.Called(ctx, containerID, options)

	return args.Error(0)
}

func (m *MockDocker) ContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, containerID, options)

	rc, _ := args.Get(0).(io.ReadCloser)

	return rc, args.Error(1)
}

func (m *MockDocker) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)

	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockDocker) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, container, config)

	if idr, ok := args.Get(0).(types.IDResponse); ok {
		return idr, args.Error(1)
	}

	return types.IDResponse{}, args.Error(1)
}

func (m *MockDocker) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	args := m.Called(ctx, execID, config)

	return args.Error(0)
}

func (m *MockDocker) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)

	if hjr, ok := args.Get(0).(types.HijackedResponse); ok {
		return hjr, args.Error(1)
	}

	return types.HijackedResponse{}, args.Error(1)
}

func (m *MockDocker) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)

	if idr, ok := args.Get(0).(types.ContainerExecInspect); ok {
		return idr, args.Error(1)
	}

	return types.ContainerExecInspect{}, args.Error(1)
}

func (m *MockDocker) ContainerExecResize(ctx context.Context, execID string, config types.ResizeOptions) error {
	args := m.Called(ctx, execID, config)

	return args.Error(0)
}

func (m *MockDocker) CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	args := m.Called(ctx, containerID, srcPath)

	rc, _ := args.Get(0).(io.ReadCloser)
	t, ok := args.Get(1).(types.ContainerPathStat)
	if !ok {
		t = types.ContainerPathStat{}
	}

	return rc, t, args.Error(2)
}

func (m *MockDocker) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	args := m.Called(ctx, container, path, content, options)

	return args.Error(0)
}

func (m *MockDocker) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	args := m.Called(ctx, options)

	if n, ok := args.Get(0).([]types.NetworkResource); ok {
		return n, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockDocker) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	args := m.Called(ctx, name, options)

	if n, ok := args.Get(0).(types.NetworkCreateResponse); ok {
		return n, args.Error(1)
	}

	return types.NetworkCreateResponse{}, args.Error(1)
}

func (m *MockDocker) NetworkRemove(ctx context.Context, networkID string) error {
	args := m.Called(ctx, networkID)

	return args.Error(0)
}

func (m *MockDocker) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	args := m.Called(ctx, networkID, containerID, config)

	return args.Error(0)
}

func (m *MockDocker) NetworkDisconnect(ctx context.Context, networkID, containerID string, force bool) error {
	args := m.Called(ctx, networkID, containerID, force)

	return args.Error(0)
}

func (m *MockDocker) VolumeCreate(ctx context.Context, options volumetypes.VolumeCreateBody) (types.Volume, error) {
	args := m.Called(ctx, options)

	if v, ok := args.Get(0).(types.Volume); ok {
		return v, args.Error(1)
	}

	return types.Volume{}, args.Error(1)
}

func (m *MockDocker) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	args := m.Called(ctx, volumeID, force)

	return args.Error(0)
}

func (m *MockDocker) VolumeList(ctx context.Context, filters filters.Args) (volume.VolumeListOKBody, error) {
	args := m.Called(ctx, filters)

	if vl, ok := args.Get(0).(volume.VolumeListOKBody); ok {
		return vl, args.Error(1)
	}

	return volumetypes.VolumeListOKBody{}, args.Error(1)
}

func (m *MockDocker) ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	args := m.Called(ctx, imageIDs)

	if rc, ok := args.Get(0).(io.ReadCloser); ok {
		return rc, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockDocker) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	args := m.Called(ctx, options)

	if is, ok := args.Get(0).([]types.ImageSummary); ok {
		return is, args.Error(1)
	}

	return []types.ImageSummary{}, args.Error(1)
}

func (m *MockDocker) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	args := m.Called(ctx, imageID, options)

	if is, ok := args.Get(0).([]types.ImageDeleteResponseItem); ok {
		return is, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockDocker) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	args := m.Called(ctx, buildContext, options)

	if ibr, ok := args.Get(0).(types.ImageBuildResponse); ok {
		return ibr, args.Error(1)
	}

	return types.ImageBuildResponse{}, args.Error(1)
}

func (m *MockDocker) ServerVersion(ctx context.Context) (types.Version, error) {
	args := m.Called(ctx)

	if ver, ok := args.Get(0).(types.Version); ok {
		return ver, args.Error(1)
	}

	return types.Version{}, args.Error(1)
}
