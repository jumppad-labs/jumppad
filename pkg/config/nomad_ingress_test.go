package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNomadIngressCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadIngressDefault)

	cl, err := c.FindResource("nomad_ingress.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNomadIngress, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNomadIngressSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadIngressDisabled)

	cl, err := c.FindResource("nomad_ingress.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const nomadIngressDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_ingress "test" {
	cluster = "nomad_cluster.dc1"
	job = "a"
	group = "b"
	task = "c"
}
`

const nomadIngressDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_ingress "test" {
	disabled = true
	cluster = "nomad_cluster.dc1"
	
	job = "a"
	group = "b"
	task = "c"
}
`
