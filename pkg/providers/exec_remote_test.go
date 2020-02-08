package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testRemoteExecSetupMocks() *mocks.MockContainerTasks {
	md := &mocks.MockContainerTasks{}
	md.On("CreateContainer", mock.Anything).Return("1234", nil)
	md.On("FindContainerIDs", mock.Anything).Return([]string{"1234"}, nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"1234"}, nil)

	return md
}

func TestRemoteExecThrowsErrorIfScript(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	cc.Script = "./script.sh"
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecCreatesContainerWhenNoTarget(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateContainer", mock.Anything)
}

func TestRemoteExecCreatesContainerFailsReturnError(t *testing.T) {
	md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Return("", fmt.Errorf("boom"))

	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecWithTargetLooksupID(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	cc.TargetRef = &config.Container{Name: "test", NetworkRef: &config.Network{Name: "cloud"}}
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", "test", "cloud")
}

func TestRemoteExecWithTargetLooksupIDNotFoundReturnsError(t *testing.T) {
	md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", "test", "cloud").Return([]string{}, nil)
	cc := testRemoteExecConfig
	cc.TargetRef = &config.Container{Name: "test", NetworkRef: &config.Network{Name: "cloud"}}
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecExecutesCommand(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything)

	params := getCalls(&md.Mock, "ExecuteCommand")[0].Arguments[1].([]string)
	assert.Equal(t, testRemoteExecConfig.Command, params[0])
	assert.Equal(t, testRemoteExecConfig.Arguments[0], params[1])
	assert.Equal(t, testRemoteExecConfig.Arguments[1], params[2])
}

func TestRemoteExecExecutesCommandFailReturnsError(t *testing.T) {
	md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "ExecuteCommand")
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecRemovesContainer(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", "1234")
}

func TestRemoteExecRemoveContainerFailReturnsError(t *testing.T) {
	md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "RemoveContainer")
	md.On("RemoveContainer", "1234").Return(fmt.Errorf("boom"))

	cc := testRemoteExecConfig
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecDoesNOTRemovesContainerWhenTarget(t *testing.T) {
	md := testRemoteExecSetupMocks()
	cc := testRemoteExecConfig
	cc.Target = "container.test"
	p := NewRemoteExec(cc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything)
}

var testRemoteExecConfig = config.RemoteExec{
	Image:     &config.Image{Name: "tools:v1"},
	WANRef:    &config.Network{Name: "WAN"},
	Command:   "tail",
	Arguments: []string{"-f", "/dev/null"},
}
