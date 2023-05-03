package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupState(state string) func() {
	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	// write the state file
	if state != "" {
		os.MkdirAll(utils.StateDir(), os.ModePerm)
		f, err := os.Create(utils.StatePath())
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.WriteString(state)
		if err != nil {
			panic(err)
		}
	}

	return func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
}

func setupExec(state string) (*cobra.Command, *clients.MockContainerTasks, func()) {
	mt := &clients.MockContainerTasks{}
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)
	mt.On("CreateShell", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mt.On("CreateContainer", mock.Anything).Return("123", nil)
	mt.On("RemoveContainer", mock.Anything, mock.Anything).Return(nil)
	mt.On("PullImage", config.Image{Name: "shipyardrun/ingress:latest"}, false).Return(nil)

	return newExecCmd(mt), mt, setupState(state)
}

func TestExecWithInvalidResourceReturnsError(t *testing.T) {
	c, _, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"container.consulate"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestExecWithNoRunningResourceReturnsError(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	removeOn(&mt.Mock, "FindContainerIDs")
	mt.On("FindContainerIDs", "consulate", "container").Return([]string{}, nil)

	c.SetArgs([]string{"container.consulate"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestExecCreatesShellInContainer(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"container.consul"})

	err := c.Execute()
	assert.NoError(t, err)

	call := getCalls(&mt.Mock, "CreateShell")[0]

	assert.Equal(t, []string{"sh"}, call.Arguments[1].([]string))
}

func TestExecCreatesShellInContainerWithCustomCommand(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"container.consul", "--", "ls", "-las"})

	err := c.Execute()
	assert.NoError(t, err)

	call := getCalls(&mt.Mock, "CreateShell")[0]

	assert.Equal(t, []string{"ls", "-las"}, call.Arguments[1].([]string))
}

func TestExecK8sWithNoPodReturnsError(t *testing.T) {
	c, _, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestExecK8sPullsImage(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "PullImage", config.Image{Name: "shipyardrun/ingress:latest"}, false)
}

func TestExecK8sPullImageFailReturnsError(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()
	removeOn(&mt.Mock, "PullImage")

	mt.On("PullImage", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestExecK8sCreatesContainer(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "CreateContainer", mock.Anything)

	cc := getCalls(&mt.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check name starts with exec
	assert.Contains(t, cc.Name, "exec-")

	// check attaches to the correct network
	assert.Equal(t, config.NetworkAttachment{Name: "network.dc1"}, cc.Networks[0])

	// check mounts correct volumes
	wd, _ := os.Getwd()
	assert.Equal(t, config.Volume{Source: wd, Destination: "/files"}, cc.Volumes[0])
	assert.Equal(t, config.Volume{Source: utils.ShipyardHome(), Destination: "/root/.shipyard"}, cc.Volumes[1])

	// check set the environment variable for the kubeconfig
	assert.Equal(t, config.KV{Key: "KUBECONFIG", Value: "/root/.shipyard/config/k3s/kubeconfig-docker.yaml"}, cc.Environment[0])
}

func TestExecK8sCreateContainerErrorReturnsError(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()
	removeOn(&mt.Mock, "CreateContainer")
	mt.On("CreateContainer", mock.Anything).Return("", fmt.Errorf("boom"))

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestExecK8sCallsRemove(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "RemoveContainer", mock.Anything, true)
}

func TestExecCreatesShellInCluster(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.NoError(t, err)

	call := getCalls(&mt.Mock, "CreateShell")[0]

	assert.Equal(t, []string{"kubectl", "exec", "-ti", "mypod", "sh"}, call.Arguments[1].([]string))
}

func TestExecCreatesShellInClusterWithCustomCommand(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod", "--", "ls", "-las"})

	err := c.Execute()
	assert.NoError(t, err)

	call := getCalls(&mt.Mock, "CreateShell")[0]

	assert.Equal(t, []string{"kubectl", "exec", "-ti", "mypod", "ls", "-las"}, call.Arguments[1].([]string))
}

func TestExecCreatesShellErrorReturnsError(t *testing.T) {
	c, mt, cleanup := setupExec(baseState)
	defer cleanup()
	removeOn(&mt.Mock, "CreateShell")

	mt.On("CreateShell", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	c.SetArgs([]string{"k8s_cluster.k3s", "mypod"})

	err := c.Execute()
	assert.Error(t, err)
}

var baseState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "running",
      "subnet": "10.15.0.0/16",
      "type": "network"
	},
	{
      "name": "k3s",
      "status": "running",
	  "type": "k8s_cluster",
	  "networks": [{
		"name": "network.dc1"
	  }]
	},
	{
      "name": "consul",
      "status": "running",
	  "type": "container",
	  "networks": [{
		"name": "network.dc1"
	  }]
	}
  ]
}
`
