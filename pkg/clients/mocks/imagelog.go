package mocks

import (
	"github.com/stretchr/testify/mock"
)

type ImageLog struct {
	mock.Mock
}

func (i *ImageLog) Log(n, t string) error {
	return i.Called(n, t).Error(0)
}

func (i *ImageLog) Read(t string) ([]string, error) {
	args := i.Called(t)

	return args.Get(0).([]string), args.Error(1)
}

func (i *ImageLog) Clear() error {
	return i.Called().Error(0)
}
