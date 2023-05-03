package providers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func testLocalExecSetupMocks() (*config.ExecLocal, *clients.CommandMock) {
	el := *execLocalConfig
	mc := &clients.CommandMock{}
	mc.On("Execute", mock.Anything).Return(123, nil)
	mc.On("Kill", mock.Anything).Return(nil)

	return &el, mc
}

func TestExecLocalExecutesCommandSuccessfully(t *testing.T) {
	c, mc := testLocalExecSetupMocks()

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Create()
	assert.NoError(t, err)

	mc.AssertCalled(t, "Execute", mock.Anything)

	params := mc.Calls[0].Arguments[0].(clients.CommandConfig)
	assert.Equal(t, c.Command, params.Command)
	assert.Equal(t, c.Arguments, params.Args)
	assert.Equal(t, c.WorkingDirectory, params.WorkingDirectory)
	assert.Equal(t, []string{"abc=123"}, params.Env)
	assert.Equal(t, c.Daemon, params.RunInBackground)

	// set log dir and pid
	assert.Equal(t, filepath.Join(utils.LogsDir(), "exec_test.log"), params.LogFilePath)
}

func TestExecLocalExecutesCommandAndSetsPid(t *testing.T) {
	c, mc := testLocalExecSetupMocks()

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Create()
	assert.NoError(t, err)

	assert.Equal(t, 123, c.Pid)
}

func TestExecLocalExecuteFailsReturnsError(t *testing.T) {
	c, mc := testLocalExecSetupMocks()

	removeOn(&mc.Mock, "Execute")
	mc.On("Execute", mock.Anything, mock.Anything).Return(0, fmt.Errorf("boom"))

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Create()
	assert.Error(t, err)
}

func TestExecLocalDestroyCallsStopWhenDaemon(t *testing.T) {
	c, mc := testLocalExecSetupMocks()
	c.Pid = 123

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Destroy()
	assert.NoError(t, err)

	mc.AssertCalled(t, "Kill", 123)
}

func TestExecLocalDestroyNotCallsStopWhenNotDaemon(t *testing.T) {
	c, mc := testLocalExecSetupMocks()
	c.Pid = 123
	c.Daemon = false

	p := NewExecLocal(c, mc, hclog.Default())

	err := p.Destroy()
	assert.NoError(t, err)

	mc.AssertNotCalled(t, "Kill", mock.Anything)
}

var execLocalConfig = &config.ExecLocal{
	ResourceInfo:     config.ResourceInfo{Name: "test", Type: config.TypeExecLocal},
	Command:          "mycommand",
	Arguments:        []string{"foo", "bar"},
	EnvVar:           map[string]string{"abc": "123"},
	Daemon:           true,
	WorkingDirectory: "./",
}
