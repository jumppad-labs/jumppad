package container

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	cMocks "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupShellMocks(t *testing.T) (*DockerTasks, *mocks.Docker) {
	md := &mocks.Docker{}
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

	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &cMocks.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	p := NewDockerTasks(md, mic, &clients.TarGz{}, clients.NewTestLogger(t))

	return p, md
}

func TestCreateShellCreatesExec(t *testing.T) {
	p, md := setupShellMocks(t)
	in := ioutil.NopCloser(bytes.NewReader([]byte("abc")))
	out := ioutil.Discard
	errW := ioutil.Discard

	err := p.CreateShell("abc", []string{"sh"}, in, out, errW)
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerExecCreate", mock.Anything, "abc", mock.Anything)

}
