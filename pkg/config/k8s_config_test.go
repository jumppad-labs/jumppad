package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesK8sConfig(t *testing.T) {
	c := NewK8sConfig("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeK8sConfig, c.Type)
}

func TestK8sConfigCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, k8sConfigValid)
	defer cleanup()

	cc, err := c.FindResource("k8s_config.test")
	assert.NoError(t, err)

	assert.Equal(t, "test", cc.Info().Name)
	assert.Equal(t, TypeK8sConfig, cc.Info().Type)
	assert.Equal(t, PendingCreation, cc.Info().Status)

	assert.Equal(t, "/tmp/files", cc.(*K8sConfig).Paths[0])
	assert.True(t, cc.(*K8sConfig).WaitUntilReady)
}

func TestK8sConfigSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, k8sConfigDisabled)
	defer cleanup()

	cc, err := c.FindResource("k8s_config.test")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cc.Info().Status)
}

func TestMakesPathAbsolute(t *testing.T) {
	c, base, cleanup := setupTestConfig(t, k8sConfigValid)
	defer cleanup()

	kc, err := c.FindResource("k8s_config.test")
	assert.NoError(t, err)

	assert.Contains(t, kc.(*K8sConfig).Paths[1], base)
}

var k8sConfigValid = `
k8s_cluster "cloud" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network {
	  name = "network.k8s"
  }
}

k8s_config "test" {
	cluster = "cluster.cloud"
	paths = ["/tmp/files","./myfiles"]
	wait_until_ready = true

	health_check {
		timeout = "30s"
		http = "http://www.google.com"
	}
}
`
var k8sConfigDisabled = `
k8s_cluster "cloud" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network {
	  name = "network.k8s"
  }
}

k8s_config "test" {
	disabled = true

	cluster = "cluster.cloud"
	paths = ["/tmp/files","./myfiles"]
	wait_until_ready = true

	health_check {
		timeout = "30s"
		http = "http://www.google.com"
	}
}
`
