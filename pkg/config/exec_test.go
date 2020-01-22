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

func TestLocalExecCreatesCorrectly(t *testing.T) {
	c, dir, cleanup := setupTestConfig(t, execLocalRelative)
	defer cleanup()

	assert.Len(t, c.LocalExecs, 1)

	ex := c.LocalExecs[0]
	assert.Equal(t, "setup_vault", ex.Name)
	assert.Equal(t, dir+"/scripts/setup_vault.sh", ex.Script)
}

func TestRemoteExecCreatesCorrectly(t *testing.T) {
	c, dir, cleanup := setupTestConfig(t, execRemoteRelative)
	defer cleanup()

	assert.Len(t, c.RemoteExecs, 1)

	ex := c.RemoteExecs[0]
	assert.Equal(t, "setup_vault", ex.Name)
	assert.Equal(t, "hashicorp/vault:latest", ex.Image.Name)
	assert.Equal(t, dir+"/scripts/setup_vault.sh", ex.Script)

	assert.Len(t, ex.Volumes, 1)
	assert.Equal(t, dir+"/scripts", ex.Volumes[0].Source)
}

var execLocalRelative = `
local_exec "setup_vault" {
  script = "./scripts/setup_vault.sh"
}
`

var execRemoteRelative = `
remote_exec "setup_vault" {
  image {
	  name = "hashicorp/vault:latest"
  }
  script = "./scripts/setup_vault.sh"
  volume {
	  source = "./scripts"
	  destination = "/files"
  }
}
`
