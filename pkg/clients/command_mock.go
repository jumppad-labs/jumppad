package clients

import (
	"github.com/stretchr/testify/mock"
)

// Can't believe it took this long to hit this circular dependency
// need to refactor away all the mocks package
type CommandMock struct {
	mock.Mock
}

func (m *CommandMock) Execute(config CommandConfig) (int, error) {
	args := m.Called(config)

	return args.Int(0), args.Error(1)
}

func (m *CommandMock) Kill(pid int) error {
	args := m.Called(pid)

	return args.Error(0)
}
