package container

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Docker defines an interface for a Docker client
//
//go:generate mockery --name Docker --filename docker.go
type Docker interface {
	ContainerCreate(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		platform *specs.Platform,
		containerName string,
	) (container.CreateResponse, error)

	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerStart(context.Context, string, types.ContainerStartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error
	ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	ContainerExecResize(ctx context.Context, execID string, config types.ResizeOptions) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)

	CheckpointCreate(ctx context.Context, container string, options types.CheckpointCreateOptions) error
	CheckpointList(ctx context.Context, container string, options types.CheckpointListOptions) ([]types.Checkpoint, error)

	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error
	CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)

	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	NetworkInspect(ctx context.Context, networkID string, options types.NetworkInspectOptions) (types.NetworkResource, error)

	NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	NetworkRemove(ctx context.Context, networkID string) error
	NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error
	NetworkDisconnect(ctx context.Context, networkID, containerID string, force bool) error

	VolumeList(ctx context.Context, opts volume.ListOptions) (volume.ListResponse, error)
	VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error

	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
	ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error)
	ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageTag(ctx context.Context, source, target string) error
	ImagePush(ctx context.Context, image string, options types.ImagePushOptions) (io.ReadCloser, error)

	ServerVersion(ctx context.Context) (types.Version, error)
	Info(ctx context.Context) (types.Info, error)
}

// NewDocker creates a new Docker client
func NewDocker() (Docker, error) {
	cli, err := client.NewClientWithOpts(client.WithHostFromEnv(), client.WithVersion("1.41"))
	if err != nil {
		return nil, err
	}

	return cli, nil
}
