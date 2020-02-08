package clients

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/mock"
)

func TestContainerRemoveCallsRemove(t *testing.T) {
	md := &mocks.MockDocker{}
	dt := NewDockerTasks(md, hclog.NewNullLogger())

	md.On("ContainerRemove", mock.Anything, "test", types.ContainerRemoveOptions{Force: true, RemoveVolumes: true}).Return(nil)

	dt.RemoveContainer("test")

	md.AssertNumberOfCalls(t, "ContainerRemove", 1)
}
