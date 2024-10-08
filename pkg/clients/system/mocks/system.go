// Code generated by mockery v2.42.3. DO NOT EDIT.

package mocks

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// System is an autogenerated mock type for the System type
type System struct {
	mock.Mock
}

// CheckVersion provides a mock function with given fields: _a0
func (_m *System) CheckVersion(_a0 string) (string, bool) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for CheckVersion")
	}

	var r0 string
	var r1 bool
	if rf, ok := ret.Get(0).(func(string) (string, bool)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) bool); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// OpenBrowser provides a mock function with given fields: _a0
func (_m *System) OpenBrowser(_a0 string) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for OpenBrowser")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Preflight provides a mock function with given fields:
func (_m *System) Preflight() (string, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Preflight")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func() (string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PromptInput provides a mock function with given fields: in, out, message
func (_m *System) PromptInput(in io.Reader, out io.Writer, message string) string {
	ret := _m.Called(in, out, message)

	if len(ret) == 0 {
		panic("no return value specified for PromptInput")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func(io.Reader, io.Writer, string) string); ok {
		r0 = rf(in, out, message)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewSystem creates a new instance of System. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSystem(t interface {
	mock.TestingT
	Cleanup(func())
}) *System {
	mock := &System{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
