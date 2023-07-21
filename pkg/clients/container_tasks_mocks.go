package clients

import (
	"io"

	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/stretchr/testify/mock"
)

type MockContainerTasks struct {
	mock.Mock
}

func (m *MockContainerTasks) SetForcePull(f bool) {
	m.Called(f)
}

func (m *MockContainerTasks) CreateContainer(c *resources.Container) (id string, err error) {
	args := m.Called(c)

	return args.String(0), args.Error(1)
}

func (m *MockContainerTasks) ContainerInfo(id string) (interface{}, error) {
	args := m.Called(id)

	return args.Get(0), args.Error(1)
}

func (m *MockContainerTasks) RemoveContainer(id string, force bool) error {
	args := m.Called(id, force)

	return args.Error(0)
}

func (m *MockContainerTasks) BuildContainer(config *resources.Container, force bool) (string, error) {
	args := m.Called(config, force)
	return args.String(0), args.Error(1)
}

func (m *MockContainerTasks) CreateVolume(name string) (id string, err error) {
	args := m.Called(name)

	return args.String(0), args.Error(1)
}

func (m *MockContainerTasks) RemoveVolume(name string) error {
	args := m.Called(name)

	return args.Error(0)
}

func (m *MockContainerTasks) PullImage(i resources.Image, f bool) error {
	args := m.Called(i, f)

	return args.Error(0)
}

func (m *MockContainerTasks) FindContainerIDs(fqdn string) ([]string, error) {
	args := m.Called(fqdn)

	if sa, ok := args.Get(0).([]string); ok {
		return sa, args.Error(1)
	}

	return nil, args.Error(1)
}



func (d *MockContainerTasks) ContainerLogs(id string, stdOut, stdErr bool) (io.ReadCloser, error) {
	args := d.Called(id, stdOut, stdErr)

	if rc, ok := args.Get(0).(io.ReadCloser); ok {
		return rc, args.Error(1)
	}

	return nil, args.Error(1)
}

func (d *MockContainerTasks) CopyFromContainer(id, src, dst string) error {
	args := d.Called(id, src, dst)

	return args.Error(0)
}

func (d *MockContainerTasks) CopyFileToContainer(id, src, dst string) error {
	args := d.Called(id, src, dst)

	return args.Error(0)
}

func (d *MockContainerTasks) CopyLocalDockerImagesToVolume(images []string, volume string, force bool) ([]string, error) {
	args := d.Called(images, volume, force)

	if a, ok := args.Get(0).([]string); ok {
		return a, args.Error(1)
	}

	return nil, args.Error(1)
}

func (d *MockContainerTasks) CopyFilesToVolume(volume string, files []string, path string, force bool) ([]string, error) {
	args := d.Called(volume, files, path, force)

	if a, ok := args.Get(0).([]string); ok {
		return a, args.Error(1)
	}

	return nil, args.Error(1)
}

func (d *MockContainerTasks) ExecuteCommand(id string, command []string, env []string, workingDirectory string, user, group string, timeout int, writer io.Writer) (int, error) {
	args := d.Called(id, command, env, workingDirectory, user, group, writer)

	return args.Int(0), args.Error(1)
}

func (d *MockContainerTasks) DetachNetwork(network, containerid string) error {
	args := d.Called(network, containerid)

	return args.Error(0)
}

func (d *MockContainerTasks) AttachNetwork(network, containerid string, aliases []string, ipaddress string) error {
	args := d.Called(network, containerid, aliases, ipaddress)

	return args.Error(0)
}

func (d *MockContainerTasks) ListNetworks(id string) []resources.NetworkAttachment {
	args := d.Called(id)

	if n, ok := args.Get(0).([]resources.NetworkAttachment); ok {
		return n
	}

	return nil
}

func (d *MockContainerTasks) CreateShell(id string, command []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error {
	args := d.Called(id, command, stdin, stdout, stderr)

	return args.Error(0)
}

func (d *MockContainerTasks) EngineInfo() *EngineInfo {
	args := d.Called()

	if ei, ok := args.Get(0).(*EngineInfo); ok {
		return ei
	}

	return nil
}
