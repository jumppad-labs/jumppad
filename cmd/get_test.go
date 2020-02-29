package cmd

import (
	"fmt"
	"testing"

	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupGet(t *testing.T) (*cobra.Command, *mocks.Blueprints) {
	bp := &mocks.Blueprints{}
	bp.On("Get", mock.Anything, mock.Anything).Return(nil)

	return newGetCmd(bp), bp
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
