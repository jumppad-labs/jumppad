package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/mock"
)

type MockContainerTasks struct {
	mock.Mock
}

func (m *MockContainerTasks) CreateContainer(c config.Container) (id string, err error) {
	args := m.Called(c)

	return args.String(0), args.Error(1)
}

func (m *MockContainerTasks) PullImage(i config.Image, f bool) error {
	args := m.Called(i, f)

	return args.Error(0)
}

func (m *MockContainerTasks) FindContainerIDs(name string, networkName string) ([]string, error) {
	args := m.Called(name, networkName)

	if sa, ok := args.Get(0).([]string); ok {
		return sa, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockContainerTasks) RemoveContainer(id string) error {
	args := m.Called(id)

	return args.Error(0)
}
