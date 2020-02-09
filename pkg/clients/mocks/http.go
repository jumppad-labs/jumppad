package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// MockHTTP is a mock implementation of the HTTP client
// interface
type MockHTTP struct {
	mock.Mock
}

func (m *MockHTTP) HealthCheckHTTP(uri string, timeout time.Duration) error {
	args := m.Called(uri, timeout)

	return args.Error(0)
}

func (m *MockHTTP) HealthCheckNomad(api_addr string, nodeCount int, timeout time.Duration) error {
	args := m.Called(api_addr, nodeCount, timeout)

	return args.Error(0)
}
