package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockHelm struct {
	mock.Mock
}

func (h*MockHelm) Create(kubeConfig, name, chartPath, valuesPath string) error {
	args :=  h.Called(kubeConfig, name,chartPath,valuesPath)	

	return args.Error(0)
}