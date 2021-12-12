package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesSidecar(t *testing.T) {
	c := NewSidecar("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeSidecar, c.Type)
}

func TestSidecarCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, sidecarDefault)

	cl, err := c.FindResource("sidecar.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeSidecar, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestSidecarSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, sidecarDisabled)

	cl, err := c.FindResource("sidecar.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const sidecarDefault = `
sidecar "test" {
	target = "container.test"
	image {
		name = "consul"
	}
}
`

const sidecarDisabled = `
sidecar "test" {
	disabled = true
	target = "container.test"
	image {
		name = "consul"
	}
}
`
