package container

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	imocks "github.com/jumppad-labs/jumppad/pkg/clients/images/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
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
	md.On("ContainerExecInspect", mock.Anything, mock.Anything).Return(container.ExecInspect{ExitCode: 0}, nil)

	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &imocks.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))
	p.defaultWait = 1 * time.Millisecond
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
