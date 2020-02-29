package mocks

import (
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/mock"
)

// Engine is a mock engine which can be used when testing the
// CLI commands
type Engine struct {
	mock.Mock
}

func (e *Engine) Apply(path string) error {
	args := e.Called(path)

	return args.Error(0)
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
