package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesModule(t *testing.T) {
	c := NewModule("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeModule, c.Type)
}

func TestModuleCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, moduleDefault)

	_, err := c.FindResource("module.testing")
	assert.Error(t, err) // should not add a resource

}

const moduleDefault = `
module "testing" {
	source = "../../examples/single_file"
}
`
