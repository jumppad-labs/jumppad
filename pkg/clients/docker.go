package clients

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// Docker defines an interface for a Docker client
type Docker interface {
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)

	ContainerCreate(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		containerName string,
	) (container.ContainerCreateCreatedBody, error)
	ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerStart(context.Context, string, types.ContainerStartOptions) error
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)

	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error
	CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error)

	NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error)
	NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error)
	NetworkRemove(ctx context.Context, networkID string) error

	VolumeCreate(ctx context.Context, options volumetypes.VolumeCreateBody) (types.Volume, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error

	ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error)
}

// NewDocker creates a new Docker client
func NewDocker() (Docker, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return cli, nil
}
