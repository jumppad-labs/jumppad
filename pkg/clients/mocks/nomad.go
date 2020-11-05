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

func (m *MockNomad) Create(files []string) error {
	args := m.Called(files)

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

func (m *MockNomad) JobRunning(job string) (bool, error) {
	args := m.Called(job)

	return args.Bool(0), args.Error(1)
}

func (m *MockNomad) Endpoints(job, group, task string) ([]map[string]string, error) {
	args := m.Called(job, group, task)

	if a, ok := args.Get(0).([]map[string]string); ok {
		return a, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockNomad) HealthCheckAPI(timeout time.Duration) error {
	args := m.Called(timeout)

	return args.Error(0)
}
