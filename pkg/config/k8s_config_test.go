package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupK8sConfig(t *testing.T, contents string) (*Config, func()) {
	dir, cleanup := createTestFiles(t)
	createNamedFile(t, dir, "*.hcl", contents)

	c := &Config{}
	err := ParseFolder(dir, c)
	assert.NoError(t, err)

	err = ParseReferences(c)
	assert.NoError(t, err)

	return c, cleanup
}

func TestCreatesCorrectly(t *testing.T) {
	c, cleanup := setupK8sConfig(t, k8sConfigValid)
	defer cleanup()

	assert.Len(t, c.K8sConfig, 1)

	k8s := c.K8sConfig[0]
	assert.Equal(t, c.Clusters[0], k8s.ClusterRef)
	assert.Equal(t, "/tmp/files", k8s.Paths[0])
	assert.True(t, k8s.WaitUntilReady)
}

func TestMakesPathAbsolute(t *testing.T) {
	c, cleanup := setupK8sConfig(t, k8sConfigValid)
	defer cleanup()

	assert.Contains(t, c.K8sConfig[0].Paths[1], "/var/folders/")
}

var k8sConfigValid = `
cluster "cloud" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network = "network.k8s"
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
