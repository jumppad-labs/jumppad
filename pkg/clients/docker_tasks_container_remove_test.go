package clients

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/mock"
)

func TestContainerRemoveCallsRemoveGently(t *testing.T) {
	md := &mocks.MockDocker{}
	mic := &clients.ImageLog{}
	dt := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnStopFailForces(t *testing.T) {
	md := &mocks.MockDocker{}
	mic := &clients.ImageLog{}
	dt := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}

func TestContainerRemoveCallsRemoveGentlyOnRemoveFailForces(t *testing.T) {
	md := &mocks.MockDocker{}
	mic := &clients.ImageLog{}
	dt := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	md.On("ContainerStop", mock.Anything, "test", mock.Anything).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(fmt.Errorf("boom"))
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", false)

	md.AssertNumberOfCalls(t, "ContainerStop", 1)
	md.AssertNumberOfCalls(t, "ContainerRemove", 2)
}

func TestContainerRemoveFailsCallsRemoveForcefully(t *testing.T) {
	md := &mocks.MockDocker{}
	mic := &clients.ImageLog{}
	dt := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: false, RemoveVolumes: true}).Return(nil)
	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test", true)
	md.AssertCalled(t, "ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})

	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}
