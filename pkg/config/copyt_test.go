package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesCopy(t *testing.T) {
	c := NewCopy("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeCopy, c.Type)
}

func TestCopyCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, copy)

	cl, err := c.FindResource("copy.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeCopy, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)

	assert.Equal(t, "/", cl.(*Copy).Source)
	assert.Equal(t, "/path", cl.(*Copy).Destination)
	assert.Equal(t, "0700", cl.(*Copy).Permissions)
}

func TestCopyDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, copyDisabled)

	cl, err := c.FindResource("copy.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, Disabled, cl.Info().Status)
}

const copy = `
copy "testing" {
	source = "/"
	destination = "/path"
	permissions = "0700"
}
`

const copyDisabled = `
copy "testing" {
	disabled = true
	source = "/"
	destination = "/path"
	permissions = "0700"
}
`
