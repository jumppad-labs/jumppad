package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesNomadCluster(t *testing.T) {
	c := NewNomadCluster("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeNomadCluster, c.Type)
}

func TestNomadClusterCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadClusterDefault)

	cl, err := c.FindResource("nomad_cluster.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNomadCluster, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNomadClusterSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadClusterDisabled)

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
