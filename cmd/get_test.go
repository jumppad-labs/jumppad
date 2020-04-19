package cmd

import (
	"fmt"
	"testing"

	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupGet(t *testing.T) (*cobra.Command, *mocks.Getter) {
	bp := &mocks.Getter{}
	bp.On("Get", mock.Anything, mock.Anything).Return(nil)
	bp.On("SetForce", mock.Anything)

	return newGetCmd(bp), bp
}

func TestGetWithForceSetsForce(t *testing.T) {
	c, bp := setupGet(t)
	c.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})
	c.Flags().Set("force-update", "true")

	err := c.Execute()
	assert.NoError(t, err)

	bp.AssertCalled(t, "SetForce", true)
}

func TestGetWithNoArgsReturnsError(t *testing.T) {
	c, _ := setupGet(t)

	err := c.Execute()
	assert.Error(t, err)
}
func TestGetGetsBlueprint(t *testing.T) {
	c, bp := setupGet(t)
	c.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})

	err := c.Execute()
	assert.NoError(t, err)

	bp.AssertCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestGetWithLocalFolderReturnsError(t *testing.T) {
	c, _ := setupGet(t)
	c.SetArgs([]string{"/tmp"})

	err := c.Execute()
	assert.Error(t, err)
}

func TestGetWhenGetErrorReturnsError(t *testing.T) {
	c, bp := setupGet(t)
	c.SetArgs([]string{"/tmp"})

	removeOn(&bp.Mock, "Get")
	bp.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := c.Execute()
	assert.Error(t, err)
}
