package clients

import (
	"github.com/stretchr/testify/mock"
)

// Can't believe it took this long to hit this circular dependency
// need to refactor away all the mocks package
type CommandMock struct {
	mock.Mock
}

func (m *CommandMock) Execute(config CommandConfig) error {
	args := m.Called(config)

	return args.Error(0)
}
