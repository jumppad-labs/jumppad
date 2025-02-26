package exec

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	commandMocks "github.com/jumppad-labs/jumppad/pkg/clients/command/mocks"
	cmdTypes "github.com/jumppad-labs/jumppad/pkg/clients/command/types"
	containerMocks "github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupProvider(t *testing.T) (*Exec, *Provider, *commandMocks.Command, *containerMocks.ContainerTasks) {
	cm := &commandMocks.Command{}
	cm.On("Execute", mock.Anything).Return(1, nil)

	dm := &containerMocks.ContainerTasks{}
	dm.On("FindContainerIDs", mock.Anything).Return([]string{"abc123"}, nil)
	dm.On("ExecuteScript", "abc123", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	dm.On("CopyFromContainer", "abc123", mock.Anything, mock.Anything).Return(nil)
	dm.On("ExecuteCommand", "abc123", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)

	e := &Exec{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "test", ID: "resource.exec.test"}}}
	p := &Provider{config: e, log: logger.NewTestLogger(t), command: cm, container: dm}

	return e, p, cm, dm
}

func TestInjectsOutputEnvIntoLocal(t *testing.T) {
	e, p, cm, _ := setupProvider(t)
	e.Script = "echo FOO=BAR >> $EXEC_OUTPUT"

	err := p.Create(context.Background())
	require.NoError(t, err)

	ac := testutils.GetCalls(&cm.Mock, "Execute")[0].Arguments[0].(cmdTypes.CommandConfig)

	td := utils.JumppadTemp()
	require.Contains(t, ac.Env, fmt.Sprintf("EXEC_OUTPUT=%s/resource.exec.test.out", td))
}

func TestParsesOutput(t *testing.T) {
	e, p, _, _ := setupProvider(t)
	e.Script = "echo FOO=BAR >> $EXEC_OUTPUT"

	// write the output for the test

	td := utils.JumppadTemp()
	os.WriteFile(fmt.Sprintf("%s/resource.exec.test.out", td), []byte("FOO=BAR"), 0644)
	t.Cleanup(func() {
		os.Remove(fmt.Sprintf("%s/resource.exec.test.out", td))
	})

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.True(t, e.ExecOutput["FOO"] == "BAR")
	require.True(t, e.Output.AsValueMap()["FOO"].AsString() == "BAR")
}

func TestDeletesOutput(t *testing.T) {
	e, p, _, _ := setupProvider(t)
	e.Script = "echo FOO=BAR >> $EXEC_OUTPUT"

	// write the output for the test
	td := utils.JumppadTemp()
	os.WriteFile(fmt.Sprintf("%s/resource.exec.test.out", td), []byte("FOO=BAR"), 0644)

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.NoFileExists(t, fmt.Sprintf("%s/resource.exec.test.out", td))
}

func TestCopiesOutputInExec(t *testing.T) {
	c := &container.Container{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "test", ID: "container.exec.test"}}}

	e, p, _, dm := setupProvider(t)
	e.Target = c
	e.Script = "echo FOO=BAR >> $EXEC_OUTPUT"

	// write the output for the test
	td := utils.JumppadTemp()
	os.WriteFile(fmt.Sprintf("%s/resource.exec.test.out", td), []byte("FOO=BAR"), 0644)
	t.Cleanup(func() {
		os.Remove(fmt.Sprintf("%s/resource.exec.test.out", td))
	})

	err := p.Create(context.Background())
	require.NoError(t, err)

	// test copies file
	cp := testutils.GetCalls(&dm.Mock, "CopyFromContainer")[0].Arguments
	require.Equal(t, "abc123", cp[0].(string))
	require.Equal(t, "/tmp/exec.out", cp[1].(string))
	require.Equal(t, fmt.Sprintf("%s/resource.exec.test.out", td), cp[2].(string))

	// test cleans up file
	rm := testutils.GetCalls(&dm.Mock, "ExecuteCommand")[0].Arguments[1].([]string)
	require.Equal(t, []string{"rm", "/tmp/exec.out"}, rm)
}
