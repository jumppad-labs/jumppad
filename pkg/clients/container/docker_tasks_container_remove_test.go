package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/instruqt/jumppad/pkg/clients/container/mocks"
	imocks "github.com/instruqt/jumppad/pkg/clients/images/mocks"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/clients/tar"
	"github.com/stretchr/testify/mock"
)

func setupRemoveTests(t *testing.T) (*DockerTasks, *mocks.Docker) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &imocks.ImageLog{}
	dt, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	return dt, md
}

func TestContainerRemoveCallsRemoveGently(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnStopFailForces(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnRemoveFailForces(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: false, RemoveVolumes: true}).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 2)
}

func TestContainerRemoveFailsCallsRemoveForcefully(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", true)
	md.AssertCalled(t, "ContainerRemove", mock.Anything, "test", container.RemoveOptions{Force: true, RemoveVolumes: true})

	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}
