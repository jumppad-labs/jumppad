package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesDocs(t *testing.T) {
	c := NewDocs("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeDocs, c.Type)
}

func TestDocsCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, docsDefault)
	defer cleanup()

	cl, err := c.FindResource("docs.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeDocs, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestDocsSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, docsDisabled)
	defer cleanup()

	cl, err := c.FindResource("docs.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, Disabled, cl.Info().Status)
}

const docsDefault = `
docs "testing" {
	path = "/"
	port = "80"
	index_title = "test"
	index_pages = ["test"]
}
`
const docsDisabled = `
docs "testing" {
	disabled = true

	path = "/"
	port = "80"
	index_title = "test"
	index_pages = ["test"]
}
`
