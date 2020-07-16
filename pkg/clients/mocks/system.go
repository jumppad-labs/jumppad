package mocks

import (
	"io"

	"github.com/stretchr/testify/mock"
)

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

func (b *System) CheckVersion(current string) (string, bool) {
	args := b.Called(current)
	return args.String(0), args.Bool(1)
}

func (b *System) PromptInput(in io.Reader, out io.Writer, message string) string {
	return b.Called(in, out, message).String(0)
}
