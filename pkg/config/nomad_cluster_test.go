package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNomadClusterCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, nomadClusterDefault)
	defer cleanup()

	cl, err := c.FindResource("nomad_cluster.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNomadCluster, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNomadClusterSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, nomadClusterDisabled)
	defer cleanup()

	cl, err := c.FindResource("nomad_cluster.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const nomadClusterDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_cluster "test" {
}
`

const nomadClusterDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_cluster "test" {
	disabled = true
}
`
