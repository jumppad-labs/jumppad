package mocks

import "github.com/stretchr/testify/mock"

type System struct {
	mock.Mock
}

func (b *System) OpenBrowser(uri string) error {
	return b.Called(uri).Error(0)
}

func (b *System) Preflight() (string, error) {
	args := b.Called()
	return "", args.Error(0)
}
