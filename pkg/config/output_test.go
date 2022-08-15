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
	c, _ := CreateConfigFromStrings(t, outputDefault)

	cl, err := c.FindResource("output.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeOutput, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestOutputSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, outputDisabled)

	_, err := c.FindResource("output.test")
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "unable to find resources")
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
