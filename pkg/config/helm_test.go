package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, helmDefault)
	defer cleanup()

	h, err := c.FindResource("helm.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", h.Info().Name)
	assert.Equal(t, TypeHelm, h.Info().Type)
	assert.Equal(t, PendingCreation, h.Info().Status)
}

const helmDefault = `
helm "testing" {
	cluster = "cluster.k3s"

	chart = "test"
	values = "test"
}
`
