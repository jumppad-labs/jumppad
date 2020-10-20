package providers

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testLocalExecSetupMocks() (*config.ExecLocal, *clients.CommandMock) {
	el := config.NewExecLocal("tester")
	el.Command = "mycommand"
	el.Arguments = []string{"arg1", "arg2"}
	el.WorkingDirectory = "mine"
	el.EnvVar = map[string]string{"abc": "123"}

	mc := &clients.CommandMock{}
	mc.On("Execute", mock.Anything).Return(nil)

	return el, mc
}

func TestExecLocalExecutesCommand(t *testing.T) {
	c, mc := testLocalExecSetupMocks()

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Create()
	assert.NoError(t, err)

	params := mc.Calls[0].Arguments[0].(clients.CommandConfig)
	assert.Equal(t, c.Command, params.Command)
	assert.Equal(t, c.Arguments, params.Args)
	assert.Equal(t, c.WorkingDirectory, params.WorkingDirectory)
	assert.Equal(t, []string{"abc=123"}, params.Env)
}
