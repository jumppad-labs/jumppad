package mocks

import "github.com/stretchr/testify/mock"

type Blueprints struct {
	mock.Mock
}

func (mb *Blueprints) Get(src, dst string) error {
	args := mb.Called(src, dst)
	return args.Error(0)
}
