package clients

import (
	"context"

	"github.com/docker/engine/api/types/container"
	"github.com/docker/engine/api/types/network"
)

type Docker interface {
	ContainerCreate(
		ctx context.Context,
		config *container.Config,
		hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig,
		containerName string,
	) (container.ContainerCreateCreatedBody, error)
}
