package cmd

import (
	"fmt"
	"testing"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPush(state string) (*cobra.Command, *clients.MockContainerTasks, func()) {
	mt := &clients.MockContainerTasks{}
	mt.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)
	mt.On("PullImage", mock.Anything, false).Return(nil)
	mt.On("CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything).Return([]string{"/images/file.tar"}, nil)
	mt.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mt.On("SetForcePull", mock.Anything).Return(nil)

	mk := &clients.MockKubernetes{}
	mh := &mocks.MockHTTP{}
	mn := &mocks.MockNomad{}

	return newPushCmd(mt, mk, mh, mn, clients.NewTestLogger(t)), mt, setupState(state)
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

	mt.AssertNotCalled(t, "CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushWithForceSetsFlag(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "k8s_cluster.k3s"})
	c.Flags().Set("force-update", "true")
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, true)
	mt.AssertCalled(t, "SetForcePull", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushK8sClusterPushesImage(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "k8s_cluster.k3s"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, true)
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

	mt.AssertNotCalled(t, "CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything)
}

func TestPushNomadClusterPushesImage(t *testing.T) {
	c, mt, cleanup := setupPush(clusterState)
	defer cleanup()

	c.SetArgs([]string{"consul:v1.6.1", "nomad_cluster.nomad"})
	err := c.Execute()
	assert.NoError(t, err)

	mt.AssertCalled(t, "CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything)
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
