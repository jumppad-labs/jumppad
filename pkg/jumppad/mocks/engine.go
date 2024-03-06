// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	hclconfig "github.com/jumppad-labs/hclconfig"

	mock "github.com/stretchr/testify/mock"

	types "github.com/jumppad-labs/hclconfig/types"
)

// Engine is an autogenerated mock type for the Engine type
type Engine struct {
	mock.Mock
}

// Apply provides a mock function with given fields: _a0
func (_m *Engine) Apply(_a0 string) (*hclconfig.Config, error) {
	ret := _m.Called(_a0)

	var r0 *hclconfig.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*hclconfig.Config, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) *hclconfig.Config); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hclconfig.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ApplyWithVariables provides a mock function with given fields: path, variables, variablesFile
func (_m *Engine) ApplyWithVariables(path string, variables map[string]string, variablesFile string) (*hclconfig.Config, error) {
	ret := _m.Called(path, variables, variablesFile)

	var r0 *hclconfig.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) (*hclconfig.Config, error)); ok {
		return rf(path, variables, variablesFile)
	}
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) *hclconfig.Config); ok {
		r0 = rf(path, variables, variablesFile)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hclconfig.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(string, map[string]string, string) error); ok {
		r1 = rf(path, variables, variablesFile)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Config provides a mock function with given fields:
func (_m *Engine) Config() *hclconfig.Config {
	ret := _m.Called()

	var r0 *hclconfig.Config
	if rf, ok := ret.Get(0).(func() *hclconfig.Config); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hclconfig.Config)
		}
	}

	return r0
}

// Destroy provides a mock function with given fields: force
func (_m *Engine) Destroy(force bool) error {
	ret := _m.Called(force)

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Diff provides a mock function with given fields: path, variables, variablesFile
func (_m *Engine) Diff(path string, variables map[string]string, variablesFile string) ([]types.Resource, []types.Resource, []types.Resource, *hclconfig.Config, error) {
	ret := _m.Called(path, variables, variablesFile)

	var r0 []types.Resource
	var r1 []types.Resource
	var r2 []types.Resource
	var r3 *hclconfig.Config
	var r4 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) ([]types.Resource, []types.Resource, []types.Resource, *hclconfig.Config, error)); ok {
		return rf(path, variables, variablesFile)
	}
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) []types.Resource); ok {
		r0 = rf(path, variables, variablesFile)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Resource)
		}
	}

	if rf, ok := ret.Get(1).(func(string, map[string]string, string) []types.Resource); ok {
		r1 = rf(path, variables, variablesFile)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]types.Resource)
		}
	}

	if rf, ok := ret.Get(2).(func(string, map[string]string, string) []types.Resource); ok {
		r2 = rf(path, variables, variablesFile)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).([]types.Resource)
		}
	}

	if rf, ok := ret.Get(3).(func(string, map[string]string, string) *hclconfig.Config); ok {
		r3 = rf(path, variables, variablesFile)
	} else {
		if ret.Get(3) != nil {
			r3 = ret.Get(3).(*hclconfig.Config)
		}
	}

	if rf, ok := ret.Get(4).(func(string, map[string]string, string) error); ok {
		r4 = rf(path, variables, variablesFile)
	} else {
		r4 = ret.Error(4)
	}

	return r0, r1, r2, r3, r4
}

// ParseConfig provides a mock function with given fields: _a0
func (_m *Engine) ParseConfig(_a0 string) (*hclconfig.Config, error) {
	ret := _m.Called(_a0)

	var r0 *hclconfig.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*hclconfig.Config, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) *hclconfig.Config); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hclconfig.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ParseConfigWithVariables provides a mock function with given fields: _a0, _a1, _a2
func (_m *Engine) ParseConfigWithVariables(_a0 string, _a1 map[string]string, _a2 string) (*hclconfig.Config, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *hclconfig.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) (*hclconfig.Config, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, map[string]string, string) *hclconfig.Config); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hclconfig.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(string, map[string]string, string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewEngine interface {
	mock.TestingT
	Cleanup(func())
}

// NewEngine creates a new instance of Engine. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewEngine(t mockConstructorTestingTNewEngine) *Engine {
	mock := &Engine{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
