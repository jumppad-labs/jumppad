package clients

import (
	"context"

	"github.com/docker/engine/api/types/container"
	"github.com/docker/engine/api/types/network"
	"github.com/stretchr/testify/mock"
)

type MockDocker struct {
	mock.Mock
}

func (m *MockDocker) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	containerName string,
) (container.ContainerCreateCreatedBody, error) {

	args := m.Called(ctx, config, hostConfig, networkingConfig, containerName)

	if c, ok := args.Get(0).(container.ContainerCreateCreatedBody); ok {
		return c, args.Error(1)
	}

	return container.ContainerCreateCreatedBody{}, args.Error(1)
}
