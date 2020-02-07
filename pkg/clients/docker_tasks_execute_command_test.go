package clients

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testExecCommandMockSetup() *mocks.MockDocker {
	mk := &mocks.MockDocker{}
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(types.IDResponse{ID: "abc"}, nil)
	mk.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(
		types.HijackedResponse{
			Conn: &net.TCPConn{},
			Reader: bufio.NewReader(
				bytes.NewReader([]byte("log output")),
			),
		},
		nil,
	)
	mk.On("ContainerExecStart", mock.Anything, "abc", mock.Anything).Return(nil)
	mk.On("ContainerExecInspect", mock.Anything, "abc", mock.Anything).Return(types.ContainerExecInspect{Running: false, ExitCode: 0}, nil)

	return mk
}

func TestExecuteCommandCreatesExec(t *testing.T) {
	mk := testExecCommandMockSetup()
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecCreate", mock.Anything, "testcontainer", mock.Anything)
	params := getCalls(&mk.Mock, "ContainerExecCreate")[0].Arguments[2].(types.ExecConfig)
	assert.Equal(t, params.Cmd[0], command[0])
}

func TestExecuteCommandExecFailReturnError(t *testing.T) {
	mk := testExecCommandMockSetup()
	removeOn(&mk.Mock, "ContainerExecCreate")
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.NoError(t, err)
}

func TestExecuteCommandAttachesToExec(t *testing.T) {
	mk := testExecCommandMockSetup()
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecAttach", mock.Anything, "abc", mock.Anything)
}

func TestExecuteCommandAttachFailReturnError(t *testing.T) {
	mk := testExecCommandMockSetup()
	removeOn(&mk.Mock, "ContainerExecAttach")
	mk.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(nil, fmt.Errorf("boom"))
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.Error(t, err)
}

func TestExecuteCommandStartsExec(t *testing.T) {
	mk := testExecCommandMockSetup()
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecStart", mock.Anything, "abc", mock.Anything)
}

func TestExecuteStartsFailReturnsError(t *testing.T) {
	mk := testExecCommandMockSetup()
	removeOn(&mk.Mock, "ContainerExecStart")
	mk.On("ContainerExecStart", mock.Anything, "abc", mock.Anything).Return(fmt.Errorf("boom"))
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.Error(t, err)
}

func TestExecuteCommandInspectsExecAndReturnsErrorOnFail(t *testing.T) {
	mk := testExecCommandMockSetup()
	removeOn(&mk.Mock, "ContainerExecInspect")
	mk.On("ContainerExecInspect", mock.Anything, "abc", mock.Anything).Return(types.ContainerExecInspect{Running: false, ExitCode: 1}, nil)
	md := NewDockerTasks(mk, hclog.NewNullLogger())
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	err := md.ExecuteCommand("testcontainer", command, writer)
	assert.Error(t, err)

	mk.AssertCalled(t, "ContainerExecInspect", mock.Anything, "abc", mock.Anything)
}
