package mocks

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
	c types.Resource
}

func New(c types.Resource) *MockProvider {
	return &MockProvider{c: c}
}

func (m *MockProvider) Create() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProvider) Destroy() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProvider) Changed() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockProvider) Refresh() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProvider) Lookup() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockProvider) Config() types.Resource {
	return m.c
}
