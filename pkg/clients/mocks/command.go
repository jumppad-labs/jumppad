package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/stretchr/testify/mock"
)

type Command struct {
	mock.Mock
}

func (m *Command) Execute(config clients.CommandConfig) error {
	args := m.Called(config)

	return args.Error(0)
}
