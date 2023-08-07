package container

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	cmocks "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testExecCommandSetup(t *testing.T) (*DockerTasks, *mocks.Docker, *mocks.ImageLog) {
	// we need to add the stream index (stdout) as the first byte for the hijacker
	writerOutput := []byte("log output")
	writerOutput = append([]byte{1}, writerOutput...)

	mk := &mocks.Docker{}
	mk.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	mk.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(types.IDResponse{ID: "abc"}, nil)
	mk.On("ContainerExecAttach", mock.Anything, mock.Anything, mock.Anything).Return(
		types.HijackedResponse{
			Conn: &net.TCPConn{},
			Reader: bufio.NewReader(
				bytes.NewReader(writerOutput),
			),
		},
		nil,
	)
	mk.On("ContainerExecStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.On("ContainerExecInspect", mock.Anything, mock.Anything, mock.Anything).Return(types.ContainerExecInspect{Running: false, ExitCode: 0}, nil)

	il := &cmocks.ImageLog{}

	dt := NewDockerTasks(mk, il, &clients.TarGz{}, clients.NewTestLogger(t))
	return dt, mk, il
}

func TestExecuteCommandCreatesExec(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	dt, mk, _ := testExecCommandSetup(t)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := dt.ExecuteCommand("testcontainer", command, []string{"abc=123"}, "/files", "1000", "2000", 300, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecCreate", mock.Anything, "testcontainer", mock.Anything)
	params := testutils.GetCalls(&mk.Mock, "ContainerExecCreate")[0].Arguments[2].(types.ExecConfig)

	// test the command
	assert.Equal(t, params.Cmd[0], command[0])

	// test the working directory
	assert.Equal(t, params.WorkingDir, "/files")

	// check the environment variables
	assert.Equal(t, params.Env[0], "abc=123")

	// check the user
	assert.Equal(t, params.User, "1000:2000")
}

func TestExecuteCommandExecFailReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecCreate")
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)
}

func TestExecuteCommandAttachesToExec(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecAttach", mock.Anything, "abc", mock.Anything)
}

func TestExecuteCommandAttachFailReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecAttach")
	mk.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(nil, fmt.Errorf("boom"))
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)
}

//func TestExecuteCommandStartsExec(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
//	}
//
//	mk, mic := testExecCommandMockSetup()
//	md := NewDockerTasks(mk, mic, &TarGz{}, clients.NewTestLogger(t))
//	writer := bytes.NewBufferString("")
//
//	command := []string{"ls", "-las"}
//	err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", writer)
//	assert.NoError(t, err)
//
//	mk.AssertCalled(t, "ContainerExecStart", mock.Anything, "abc", mock.Anything)
//}
//
//func TestExecuteStartsFailReturnsError(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
//	}
//
//	mk, mic := testExecCommandMockSetup()
//	testutils.RemoveOn(&mk.Mock, "ContainerExecStart")
//	mk.On("ContainerExecStart", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
//	md := NewDockerTasks(mk, mic, &TarGz{}, clients.NewTestLogger(t))
//	writer := bytes.NewBufferString("")
//
//	command := []string{"ls", "-las"}
//	err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", writer)
//	assert.Error(t, err)
//}

func TestExecuteCommandInspectsExecAndReturnsErrorOnFail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecInspect")
	mk.On("ContainerExecInspect", mock.Anything, mock.Anything, mock.Anything).Return(types.ContainerExecInspect{Running: false, ExitCode: 1}, nil)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)

	mk.AssertCalled(t, "ContainerExecInspect", mock.Anything, "abc", mock.Anything)
}
