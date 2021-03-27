package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, networkDefault)
	defer cleanup()

	cl, err := c.FindResource("network.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNetwork, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNetworkSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, networkDisabled)
	defer cleanup()

	cl, err := c.FindResource("network.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const networkDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}
`

const networkDisabled = `
network "test" {
	disabled = true
	subnet = "10.0.0.0/24"
}
`
