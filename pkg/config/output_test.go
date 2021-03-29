package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesOutput(t *testing.T) {
	c := NewOutput("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeOutput, c.Type)
}

func TestOutputCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, outputDefault)
	defer cleanup()

	cl, err := c.FindResource("output.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeOutput, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestOutputSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, outputDisabled)
	defer cleanup()

	cl, err := c.FindResource("output.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const outputDefault = `
output "test" {
	value = "abcc"
}
`

const outputDisabled = `
output "test" {
	disabled = true
	value = "abcc"
}
`
