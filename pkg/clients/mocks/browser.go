package mocks

import "github.com/stretchr/testify/mock"

type Browser struct {
	mock.Mock
}

func (b *Browser) Open(uri string) error {
	return b.Called(uri).Error(0)
}
