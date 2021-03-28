package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesContainer(t *testing.T) {
	c := NewContainer("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeContainer, c.Type)
}

func TestContainerCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, containerDefault)
	defer cleanup()

	co, err := c.FindResource("container.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", co.Info().Name)
	assert.Equal(t, TypeContainer, co.Info().Type)
	assert.Equal(t, PendingCreation, co.Info().Status)
}

func TestContainerSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, containerDisabled)
	defer cleanup()

	co, err := c.FindResource("container.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", co.Info().Name)
	assert.Equal(t, Disabled, co.Info().Status)
}

const containerDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

container "testing" {
	network {
		name = "network.test"
	}
	image {
		name = "consul"
	}
}
`

const containerDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

container "testing" {
	disabled = true

	network {
		name = "network.test"
	}
	image {
		name = "consul"
	}
}
`
