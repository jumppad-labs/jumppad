package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	cmocks "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/mock"
)

func setupRemoveTests(t *testing.T) (*DockerTasks, *mocks.Docker) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &cmocks.ImageLog{}
	dt := NewDockerTasks(md, mic, &clients.TarGz{}, clients.NewTestLogger(t))

	return dt, md
}

func TestContainerRemoveCallsRemoveGently(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnStopFailForces(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnRemoveFailForces(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 2)
}

func TestContainerRemoveFailsCallsRemoveForcefully(t *testing.T) {
	dt, md := setupRemoveTests(t)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", true)
	md.AssertCalled(t, "ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})

	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}
