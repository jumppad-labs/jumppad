package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesNomadJob(t *testing.T) {
	c := NewNomadJob("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeNomadJob, c.Type)
}

func TestNomadJobCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadJobDefault)

	cl, err := c.FindResource("nomad_job.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeNomadJob, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestNomadJobSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, nomadJobDisabled)

	cl, err := c.FindResource("nomad_job.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const nomadJobDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_job "test" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout = "60s"
    nomad_jobs = ["example_2"]
  }
}
`

const nomadJobDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

nomad_job "test" {
	disabled = true
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout = "60s"
    nomad_jobs = ["example_2"]
  }
}
`
