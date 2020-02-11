package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
	config providers.ConfigWrapper
}

func New(c providers.ConfigWrapper) *MockProvider {
	return &MockProvider{config: c}
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

func (m *MockProvider) Config() providers.ConfigWrapper {
	m.Called()

	return m.config
}

// State is the state from the config
func (m *MockProvider) State() config.State {
	args := m.Called()

	return args.Get(0).(config.State)
}
