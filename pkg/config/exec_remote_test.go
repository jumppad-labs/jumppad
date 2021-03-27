package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestConfig(t *testing.T, contents string) (*Config, string, func()) {
	dir, cleanup := createTestFiles(t)
	createNamedFile(t, dir, "*.hcl", contents)

	c := New()
	err := ParseFolder(dir, c, false, "", false, []string{}, nil, "")
	assert.NoError(t, err)

	err = ParseReferences(c)
	assert.NoError(t, err)

	return c, dir, cleanup
}

func TestExecRemoteCreatesCorrectly(t *testing.T) {
	c, dir, cleanup := setupTestConfig(t, execRemoteRelative)
	defer cleanup()

	ex, err := c.FindResource("exec_remote.setup_vault")
	assert.NoError(t, err)

	assert.Equal(t, "setup_vault", ex.Info().Name)
	assert.Equal(t, TypeExecRemote, ex.Info().Type)
	assert.Equal(t, PendingCreation, ex.Info().Status)

	assert.Equal(t, "hashicorp/vault:latest", ex.(*ExecRemote).Image.Name)

	assert.Len(t, ex.(*ExecRemote).Volumes, 1)
	assert.Equal(t, filepath.Join(dir, "/scripts"), ex.(*ExecRemote).Volumes[0].Source)
}

var execRemoteRelative = `
network "cloud" {
	subnet = "192.158.32.12"
}

exec_remote "setup_vault" {
  image {
	  name = "hashicorp/vault:latest"
  }
  network {
	  name = "network.cloud"
	}

	cmd = "/scripts/setup_vault.sh"

  volume {
	  source = "./scripts"
	  destination = "/files"
  }
}
`
