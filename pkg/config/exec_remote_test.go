package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesExecRemote(t *testing.T) {
	c := NewExecRemote("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeExecRemote, c.Type)
}

func TestExecRemoteCreatesCorrectly(t *testing.T) {
	c, dir := CreateConfigFromStrings(t, execRemoteRelative)

	ex, err := c.FindResource("exec_remote.setup_vault")
	assert.NoError(t, err)

	assert.Equal(t, "setup_vault", ex.Info().Name)
	assert.Equal(t, TypeExecRemote, ex.Info().Type)
	assert.Equal(t, PendingCreation, ex.Info().Status)

	assert.Equal(t, "hashicorp/vault:latest", ex.(*ExecRemote).Image.Name)

	assert.Len(t, ex.(*ExecRemote).Volumes, 1)
	assert.Equal(t, filepath.Join(dir, "/scripts"), ex.(*ExecRemote).Volumes[0].Source)
}

func TestExecRemoteSetsDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, execRemoteDisabled)

	ex, err := c.FindResource("exec_remote.setup_vault")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, ex.Info().Status)
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
var execRemoteDisabled = `
network "cloud" {
	subnet = "192.158.32.12"
}

exec_remote "setup_vault" {
	disabled = true

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
