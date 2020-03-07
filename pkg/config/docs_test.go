package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocsCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, docsDefault)
	defer cleanup()

	cl, err := c.FindResource("docs.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeDocs, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

const docsDefault = `
docs "testing" {
	path = "/"
	port = "80"
	index_title = "test"
	index_pages = ["test"]
}
`
