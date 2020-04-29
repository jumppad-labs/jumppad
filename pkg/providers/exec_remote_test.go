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

func testRemoteExecSetupMocks() (*config.ExecRemote, *config.Network, *mocks.MockContainerTasks) {
	md := &mocks.MockContainerTasks{}
	md.On("CreateContainer", mock.Anything).Return("1234", nil)
	md.On("PullImage", mock.Anything, mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything).Return([]string{"1234"}, nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"1234"}, nil)

	trex := &config.ExecRemote{
		Image:       &config.Image{Name: "tools:v1"},
		Networks:    []config.NetworkAttachment{config.NetworkAttachment{Name: "wan"}},
		Command:     "tail",
		Arguments:   []string{"-f", "/dev/null"},
		Environment: []config.KV{config.KV{Key: "abc", Value: "123"}},
	}

	net := config.NewNetwork("wan")

	cont := config.NewContainer("test")
	cont.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "network.wan"}}

	c := config.New()
	c.AddResource(net)
	c.AddResource(trex)
	c.AddResource(cont)

	return trex, net, md
}

/*
func TestRemoteExecThrowsErrorIfScript(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	trex.Script = "./script.sh"
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}
*/

func TestRemoteExecPullsImageWhenNoTarget(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", mock.Anything, mock.Anything)
}

func TestRemoteExecPullsImageReturnsErrorWhenError(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "PullImage")
	md.On("PullImage", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecCreatesContainerWhenNoTarget(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateContainer", mock.Anything)
}

func TestRemoteExecCreatesContainerFailsReturnError(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Return("", fmt.Errorf("boom"))

	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecWithTargetLooksupID(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	trex.Target = "container.test"
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", "test", config.TypeContainer)
}

func TestRemoteExecWithTargetLooksupIDNotFoundReturnsError(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	trex.Target = "container.test"
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", "test", config.TypeContainer).Return([]string{}, nil)
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecExecutesCommand(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	params := getCalls(&md.Mock, "ExecuteCommand")[0].Arguments[1].([]string)
	env := getCalls(&md.Mock, "ExecuteCommand")[0].Arguments[2].([]string)
	assert.Equal(t, trex.Command, params[0])
	assert.Equal(t, trex.Arguments[0], params[1])
	assert.Equal(t, trex.Arguments[1], params[2])
	assert.Contains(t, env, fmt.Sprintf("%s=%s", trex.Environment[0].Key, trex.Environment[0].Value))
}

func TestRemoteExecExecutesCommandFailReturnsError(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "ExecuteCommand")
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestRemoteExecRemovesContainer(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", "1234")
}

/*
func TestRemoteExecRemoveContainerFailReturnsError(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	removeOn(&md.Mock, "RemoveContainer")
	md.On("RemoveContainer", "1234").Return(fmt.Errorf("boom"))

	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}
*/
func TestRemoteExecDoesNOTRemovesContainerWhenTarget(t *testing.T) {
	trex, _, md := testRemoteExecSetupMocks()
	trex.Target = "container.test"
	p := NewRemoteExec(trex, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything)
}
