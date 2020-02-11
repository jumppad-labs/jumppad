package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
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
	args := m.Called()

	if cw, ok := args.Get(0).(providers.ConfigWrapper); ok {
		return cw
	}

	return providers.ConfigWrapper{}
}
