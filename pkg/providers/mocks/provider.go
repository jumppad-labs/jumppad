package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
	c config.Resource
}

func New(c config.Resource) *MockProvider {
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

func (m *MockProvider) Lookup() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockProvider) Config() config.Resource {
	return m.c
}
