package clients

import (
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
