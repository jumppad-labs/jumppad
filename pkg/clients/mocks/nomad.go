package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type MockNomad struct {
	mock.Mock
}

func (m *MockNomad) HealthCheckAPI(timeout time.Duration) error {
	args := m.Called(timeout)

	return args.Error(0)
}
