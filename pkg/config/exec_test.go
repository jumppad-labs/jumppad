package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestConfig(t *testing.T, contents string) (*Config, string, func()) {
	dir, cleanup := createTestFiles(t)
	createNamedFile(t, dir, "*.hcl", contents)

	c := &Config{}
	err := ParseFolder(dir, c)
	assert.NoError(t, err)

	err = ParseReferences(c)
	assert.NoError(t, err)

	return c, dir, cleanup
}

func TestExecCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, execRelative)
	defer cleanup()

	assert.Len(t, c.Execs, 1)

	ex := c.Execs[0]
	assert.Equal(t, "setup_vault", ex.Name)
	assert.Equal(t, "./scripts/setup_vault.sh", ex.Command)
}

var execRelative = `
exec "setup_vault" {
  cmd = "./scripts/setup_vault.sh"
}
`
