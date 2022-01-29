package clients

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupShellMocks() (*DockerTasks, *clients.MockDocker) {
	md := &clients.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(types.IDResponse{ID: "123"}, nil)
	md.On("ContainerExecAttach", mock.Anything, "123", mock.Anything).Return(
		types.HijackedResponse{
			Conn: &net.TCPConn{},
			Reader: bufio.NewReader(
				bytes.NewReader([]byte("log output")),
			),
		}, nil)
	md.On("ContainerExecInspect", mock.Anything, mock.Anything).Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	mic := &clients.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	p := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	return p, md
}

func TestCreateShellCreatesExec(t *testing.T) {
	p, md := setupShellMocks()
	in := ioutil.NopCloser(bytes.NewReader([]byte("abc")))
	out := ioutil.Discard
	errW := ioutil.Discard

	err := p.CreateShell("abc", []string{"sh"}, in, out, errW)
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerExecCreate", mock.Anything, "abc", mock.Anything)

}
