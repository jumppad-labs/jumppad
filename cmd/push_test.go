package cmd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPush(state string) (*cobra.Command, *mocks.MockContainerTasks, func()) {
	mt := &mocks.MockContainerTasks{}
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)
	mt.On("PullImage", mock.Anything, false).Return(nil)
	mt.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything).Return("", nil)
	mt.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mk := &mocks.MockKubernetes{}
	mh := &mocks.MockHTTP{}

	return newPushCmd(mt, mk, mh, hclog.NewNullLogger()), mt, setupState(state)
}

func TestPushInvalidArgsReturnsError(t *testing.T) {
	c, _, cleanup := setupPush(clusterState)
	defer cleanup()

	err := c.Execute()
	assert.Error(t, err)
}

func TestPushNoResourceReturnsError(t *testing.T) {
	c, _, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "nomad_clsuter.dev"})
	err := c.Execute()
	assert.Error(t, err)
}

func TestPushInvalidResourceReturnsError(t *testing.T) {
	c, _, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "container.dev"})
	err := c.Execute()
	assert.Error(t, err)
}

func TestPushK8sClusterIDErrorReturnsError(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	removeOn(&mt.Mock, "FindContainerIDs")
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, fmt.Errorf("boom"))

	c.SetArgs([]string{"consul:v1.6.1", "k8s_cluster.k3s"})
	err := c.Execute()
	assert.Error(t, err)
}

func TestPushK8sClusterIDNotFoundReturnsError(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	removeOn(&mt.Mock, "FindContainerIDs")
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil)

	c.SetArgs([]string{"consul:v1.6.1", "k8s_cluster.k3s"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertNotCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushK8sClusterPushesImage(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "k8s_cluster.k3s"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushNomadClusterIDErrorReturnsError(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	removeOn(&mt.Mock, "FindContainerIDs")
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, fmt.Errorf("boom"))

	c.SetArgs([]string{"consul:v1.6.1", "nomad_cluster.nomad"})
	err := c.Execute()
	assert.Error(t, err)
}

func TestPushNomadClusterIDNotFoundReturnsError(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	removeOn(&mt.Mock, "FindContainerIDs")
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil)

	c.SetArgs([]string{"consul:v1.6.1", "nomad_cluster.nomad"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertNotCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushNomadClusterPushesImage(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "nomad_cluster.nomad"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "ExecuteCommand", mock.Anything, mock.Anything, mock.Anything)
}

var clusterState = `
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
      "name": "nomad",
      "status": "running",
	  "type": "nomad_cluster",
	  "networks": [{
		"name": "network.dc1"
	  }]
	}
  ]
}
`
