package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesHelm(t *testing.T) {
	c := NewHelm("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeHelm, c.Type)
}

func TestHelmCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, helmDefault)

	h, err := c.FindResource("helm.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", h.Info().Name)
	assert.Equal(t, TypeHelm, h.Info().Type)
	assert.Equal(t, PendingCreation, h.Info().Status)
}

func TestHelmSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, helmDisabled)

	h, err := c.FindResource("helm.testing")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, h.Info().Status)
}

const helmDefault = `
helm "testing" {
	cluster = "cluster.k3s"

	chart = "test"
	values = "test"
}
`

const helmDisabled = `
helm "testing" {
	disabled = true

	cluster = "cluster.k3s"

	chart = "test"
	values = "test"
}
`
