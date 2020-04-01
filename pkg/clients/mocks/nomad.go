package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type MockNomad struct {
	mock.Mock
}

func (m *MockNomad) SetConfig(c string) error {
	args := m.Called(c)

	return args.Error(0)
}

func (m *MockNomad) Create(files []string, waitUntilReady bool) error {
	args := m.Called(files, waitUntilReady)

	return args.Error(0)
}

func (m *MockNomad) Stop(files []string) error {
	args := m.Called(files)

	return args.Error(0)
}

func (m *MockNomad) ParseJob(file string) ([]byte, error) {
	args := m.Called(file)

	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockNomad) AllocationsRunning(file string) (map[string]bool, error) {
	args := m.Called(file)

	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockNomad) HealthCheckAPI(timeout time.Duration) error {
	args := m.Called(timeout)

	return args.Error(0)
}
