package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, templateDefault)
	defer cleanup()

	cl, err := c.FindResource("template.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cl.Info().Name)
	assert.Equal(t, TypeTemplate, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestTemplateSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, templateDisabled)
	defer cleanup()

	cl, err := c.FindResource("template.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const templateDefault = `
template "test" {
	source = "./container.test"
	destination = "./container.test"
}
`

const templateDisabled = `
template "test" {
	disabled = true
	source = "./container.test"
	destination = "./container.test"
}
`
