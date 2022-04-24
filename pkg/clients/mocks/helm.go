package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockHelm struct {
	mock.Mock
}

func (h *MockHelm) Create(kubeConfig, name, namespace string, createNamespace bool, skipCRDs bool, chart, version, valuesPath string, valueString map[string]string) error {
	args := h.Called(kubeConfig, name, namespace, createNamespace, skipCRDs, chart, version, valuesPath, valueString)

	return args.Error(0)
}

func (h *MockHelm) Destroy(kubeConfig, name, namespace string) error {
	args := h.Called(kubeConfig, name, namespace)

	return args.Error(0)
}

func (h *MockHelm) UpsertChartRepository(name, url string) error {
	args := h.Called(name, url)

	return args.Error(0)
}
