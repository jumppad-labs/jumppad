package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/stretchr/testify/mock"
)

// Engine is a mock engine which can be used when testing the
// CLI commands
type Engine struct {
	mock.Mock
}

func (e *Engine) GetClients() *shipyard.Clients {
	args := e.Called()

	if e, ok := args.Get(0).(*shipyard.Clients); ok {
		return e
	}

	return nil
}

func (e *Engine) Apply(path string) ([]config.Resource, error) {
	args := e.Called(path)

	if r, ok := args.Get(0).([]config.Resource); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}

func (e *Engine) Destroy(path string, all bool) error {
	args := e.Called(path, all)

	return args.Error(0)
}
func (e *Engine) ResourceCount() int {
	return e.Called().Int(0)
}
func (e *Engine) Blueprint() *config.Blueprint {
	if bp, ok := e.Called().Get(0).(*config.Blueprint); ok {
		return bp
	}

	return nil
}
