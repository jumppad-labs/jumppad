package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
)

type MockKubernetes struct {
	mock.Mock
}

func (m *MockKubernetes) SetConfig(kubeconfig string) error {
	args := m.Called(kubeconfig)

	return args.Error(0)
}

func (m *MockKubernetes) GetPods(selector string) (*v1.PodList, error) {
	args := m.Called(selector)

	if pl, ok := args.Get(0).(*v1.PodList); ok {
		return pl, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockKubernetes) Apply(files []string, waitUntilReady bool) error {
	args := m.Called(files, waitUntilReady)

	return args.Error(0)
}

func (m *MockKubernetes) Delete(files []string) error {
	args := m.Called(files)

	return args.Error(0)
}

func (m *MockKubernetes) HealthCheckPods(selectors []string, timeout time.Duration) error {
	args := m.Called(selectors, timeout)

	return args.Error(0)
}
