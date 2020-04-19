package mocks

import "github.com/stretchr/testify/mock"

type Getter struct {
	mock.Mock
}

func (mb *Getter) Get(src, dst string) error {
	args := mb.Called(src, dst)
	return args.Error(0)
}

func (mb *Getter) SetForce(force bool) {
	mb.Called(force)
}
