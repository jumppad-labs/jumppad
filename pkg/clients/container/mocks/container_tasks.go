// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	io "io"

	types "github.com/instruqt/jumppad/pkg/clients/container/types"
	mock "github.com/stretchr/testify/mock"
)

// ContainerTasks is an autogenerated mock type for the ContainerTasks type
type ContainerTasks struct {
	mock.Mock
}

// AttachNetwork provides a mock function with given fields: network, containerid, aliases, ipaddress
func (_m *ContainerTasks) AttachNetwork(network string, containerid string, aliases []string, ipaddress string) error {
	ret := _m.Called(network, containerid, aliases, ipaddress)

	if len(ret) == 0 {
		panic("no return value specified for AttachNetwork")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, []string, string) error); ok {
		r0 = rf(network, containerid, aliases, ipaddress)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BuildContainer provides a mock function with given fields: config, force
func (_m *ContainerTasks) BuildContainer(config *types.Build, force bool) (string, error) {
	ret := _m.Called(config, force)

	if len(ret) == 0 {
		panic("no return value specified for BuildContainer")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.Build, bool) (string, error)); ok {
		return rf(config, force)
	}
	if rf, ok := ret.Get(0).(func(*types.Build, bool) string); ok {
		r0 = rf(config, force)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(*types.Build, bool) error); ok {
		r1 = rf(config, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ContainerInfo provides a mock function with given fields: id
func (_m *ContainerTasks) ContainerInfo(id string) (interface{}, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for ContainerInfo")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (interface{}, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(string) interface{}); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ContainerLogs provides a mock function with given fields: id, stdOut, stdErr
func (_m *ContainerTasks) ContainerLogs(id string, stdOut bool, stdErr bool) (io.ReadCloser, error) {
	ret := _m.Called(id, stdOut, stdErr)

	if len(ret) == 0 {
		panic("no return value specified for ContainerLogs")
	}

	var r0 io.ReadCloser
	var r1 error
	if rf, ok := ret.Get(0).(func(string, bool, bool) (io.ReadCloser, error)); ok {
		return rf(id, stdOut, stdErr)
	}
	if rf, ok := ret.Get(0).(func(string, bool, bool) io.ReadCloser); ok {
		r0 = rf(id, stdOut, stdErr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	if rf, ok := ret.Get(1).(func(string, bool, bool) error); ok {
		r1 = rf(id, stdOut, stdErr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CopyFileToContainer provides a mock function with given fields: id, src, dst
func (_m *ContainerTasks) CopyFileToContainer(id string, src string, dst string) error {
	ret := _m.Called(id, src, dst)

	if len(ret) == 0 {
		panic("no return value specified for CopyFileToContainer")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(id, src, dst)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CopyFilesToVolume provides a mock function with given fields: volume, files, path, force
func (_m *ContainerTasks) CopyFilesToVolume(volume string, files []string, path string, force bool) ([]string, error) {
	ret := _m.Called(volume, files, path, force)

	if len(ret) == 0 {
		panic("no return value specified for CopyFilesToVolume")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, []string, string, bool) ([]string, error)); ok {
		return rf(volume, files, path, force)
	}
	if rf, ok := ret.Get(0).(func(string, []string, string, bool) []string); ok {
		r0 = rf(volume, files, path, force)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string, []string, string, bool) error); ok {
		r1 = rf(volume, files, path, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CopyFromContainer provides a mock function with given fields: id, src, dst
func (_m *ContainerTasks) CopyFromContainer(id string, src string, dst string) error {
	ret := _m.Called(id, src, dst)

	if len(ret) == 0 {
		panic("no return value specified for CopyFromContainer")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(id, src, dst)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CopyLocalDockerImagesToVolume provides a mock function with given fields: images, volume, force
func (_m *ContainerTasks) CopyLocalDockerImagesToVolume(images []string, volume string, force bool) ([]string, error) {
	ret := _m.Called(images, volume, force)

	if len(ret) == 0 {
		panic("no return value specified for CopyLocalDockerImagesToVolume")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func([]string, string, bool) ([]string, error)); ok {
		return rf(images, volume, force)
	}
	if rf, ok := ret.Get(0).(func([]string, string, bool) []string); ok {
		r0 = rf(images, volume, force)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func([]string, string, bool) error); ok {
		r1 = rf(images, volume, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateContainer provides a mock function with given fields: _a0
func (_m *ContainerTasks) CreateContainer(_a0 *types.Container) (string, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for CreateContainer")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.Container) (string, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*types.Container) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(*types.Container) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateFileInContainer provides a mock function with given fields: containerID, contents, filename, path
func (_m *ContainerTasks) CreateFileInContainer(containerID string, contents string, filename string, path string) error {
	ret := _m.Called(containerID, contents, filename, path)

	if len(ret) == 0 {
		panic("no return value specified for CreateFileInContainer")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string) error); ok {
		r0 = rf(containerID, contents, filename, path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateShell provides a mock function with given fields: id, command, stdin, stdout, stderr
func (_m *ContainerTasks) CreateShell(id string, command []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error {
	ret := _m.Called(id, command, stdin, stdout, stderr)

	if len(ret) == 0 {
		panic("no return value specified for CreateShell")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []string, io.ReadCloser, io.Writer, io.Writer) error); ok {
		r0 = rf(id, command, stdin, stdout, stderr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateVolume provides a mock function with given fields: name
func (_m *ContainerTasks) CreateVolume(name string) (string, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for CreateVolume")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DetachNetwork provides a mock function with given fields: network, containerid
func (_m *ContainerTasks) DetachNetwork(network string, containerid string) error {
	ret := _m.Called(network, containerid)

	if len(ret) == 0 {
		panic("no return value specified for DetachNetwork")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(network, containerid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EngineInfo provides a mock function with no fields
func (_m *ContainerTasks) EngineInfo() *types.EngineInfo {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for EngineInfo")
	}

	var r0 *types.EngineInfo
	if rf, ok := ret.Get(0).(func() *types.EngineInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.EngineInfo)
		}
	}

	return r0
}

// ExecuteCommand provides a mock function with given fields: id, command, env, workingDirectory, user, group, timeout, writer
func (_m *ContainerTasks) ExecuteCommand(id string, command []string, env []string, workingDirectory string, user string, group string, timeout int, writer io.Writer) (int, error) {
	ret := _m.Called(id, command, env, workingDirectory, user, group, timeout, writer)

	if len(ret) == 0 {
		panic("no return value specified for ExecuteCommand")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(string, []string, []string, string, string, string, int, io.Writer) (int, error)); ok {
		return rf(id, command, env, workingDirectory, user, group, timeout, writer)
	}
	if rf, ok := ret.Get(0).(func(string, []string, []string, string, string, string, int, io.Writer) int); ok {
		r0 = rf(id, command, env, workingDirectory, user, group, timeout, writer)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(string, []string, []string, string, string, string, int, io.Writer) error); ok {
		r1 = rf(id, command, env, workingDirectory, user, group, timeout, writer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExecuteScript provides a mock function with given fields: id, contents, env, workingDirectory, user, group, timeout, writer
func (_m *ContainerTasks) ExecuteScript(id string, contents string, env []string, workingDirectory string, user string, group string, timeout int, writer io.Writer) (int, error) {
	ret := _m.Called(id, contents, env, workingDirectory, user, group, timeout, writer)

	if len(ret) == 0 {
		panic("no return value specified for ExecuteScript")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, []string, string, string, string, int, io.Writer) (int, error)); ok {
		return rf(id, contents, env, workingDirectory, user, group, timeout, writer)
	}
	if rf, ok := ret.Get(0).(func(string, string, []string, string, string, string, int, io.Writer) int); ok {
		r0 = rf(id, contents, env, workingDirectory, user, group, timeout, writer)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(string, string, []string, string, string, string, int, io.Writer) error); ok {
		r1 = rf(id, contents, env, workingDirectory, user, group, timeout, writer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindContainerIDs provides a mock function with given fields: containerName
func (_m *ContainerTasks) FindContainerIDs(containerName string) ([]string, error) {
	ret := _m.Called(containerName)

	if len(ret) == 0 {
		panic("no return value specified for FindContainerIDs")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]string, error)); ok {
		return rf(containerName)
	}
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(containerName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(containerName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindImageInLocalRegistry provides a mock function with given fields: image
func (_m *ContainerTasks) FindImageInLocalRegistry(image types.Image) (string, error) {
	ret := _m.Called(image)

	if len(ret) == 0 {
		panic("no return value specified for FindImageInLocalRegistry")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Image) (string, error)); ok {
		return rf(image)
	}
	if rf, ok := ret.Get(0).(func(types.Image) string); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(types.Image) error); ok {
		r1 = rf(image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindImagesInLocalRegistry provides a mock function with given fields: filter
func (_m *ContainerTasks) FindImagesInLocalRegistry(filter string) ([]string, error) {
	ret := _m.Called(filter)

	if len(ret) == 0 {
		panic("no return value specified for FindImagesInLocalRegistry")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]string, error)); ok {
		return rf(filter)
	}
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindNetwork provides a mock function with given fields: id
func (_m *ContainerTasks) FindNetwork(id string) (types.NetworkAttachment, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for FindNetwork")
	}

	var r0 types.NetworkAttachment
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (types.NetworkAttachment, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(string) types.NetworkAttachment); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(types.NetworkAttachment)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListNetworks provides a mock function with given fields: id
func (_m *ContainerTasks) ListNetworks(id string) []types.NetworkAttachment {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for ListNetworks")
	}

	var r0 []types.NetworkAttachment
	if rf, ok := ret.Get(0).(func(string) []types.NetworkAttachment); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.NetworkAttachment)
		}
	}

	return r0
}

// PullImage provides a mock function with given fields: image, force
func (_m *ContainerTasks) PullImage(image types.Image, force bool) error {
	ret := _m.Called(image, force)

	if len(ret) == 0 {
		panic("no return value specified for PullImage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(types.Image, bool) error); ok {
		r0 = rf(image, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PushImage provides a mock function with given fields: image
func (_m *ContainerTasks) PushImage(image types.Image) error {
	ret := _m.Called(image)

	if len(ret) == 0 {
		panic("no return value specified for PushImage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(types.Image) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveContainer provides a mock function with given fields: id, force
func (_m *ContainerTasks) RemoveContainer(id string, force bool) error {
	ret := _m.Called(id, force)

	if len(ret) == 0 {
		panic("no return value specified for RemoveContainer")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(id, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveImage provides a mock function with given fields: id
func (_m *ContainerTasks) RemoveImage(id string) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for RemoveImage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveVolume provides a mock function with given fields: name
func (_m *ContainerTasks) RemoveVolume(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for RemoveVolume")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetForce provides a mock function with given fields: _a0
func (_m *ContainerTasks) SetForce(_a0 bool) {
	_m.Called(_a0)
}

// TagImage provides a mock function with given fields: source, destination
func (_m *ContainerTasks) TagImage(source string, destination string) error {
	ret := _m.Called(source, destination)

	if len(ret) == 0 {
		panic("no return value specified for TagImage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(source, destination)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewContainerTasks creates a new instance of ContainerTasks. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewContainerTasks(t interface {
	mock.TestingT
	Cleanup(func())
}) *ContainerTasks {
	mock := &ContainerTasks{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
