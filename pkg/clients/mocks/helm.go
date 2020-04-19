package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockHelm struct {
	mock.Mock
}

func (h *MockHelm) Create(kubeConfig, name, chartPath, valuesPath string, valueString map[string]string) error {
	args := h.Called(kubeConfig, name, chartPath, valuesPath, valueString)

	return args.Error(0)
}

func (h *MockHelm) Destroy(kubeConfig, name string) error {
	args := h.Called(kubeConfig, name)

	return args.Error(0)
}
