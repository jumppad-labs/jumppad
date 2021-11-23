package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesNetwork(t *testing.T) {
	c := NewNetwork("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeNetwork, c.Type)
}

func TestNetworkCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, networkDefault)

	cl, err := c.FindResource("network.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNetwork, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNetworkSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, networkDisabled)

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
