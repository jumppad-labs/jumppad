package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesK8sIngress(t *testing.T) {
	c := NewK8sIngress("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeK8sIngress, c.Type)
}

func TestK8sIngressCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, k8sIngressDefault)
	defer cleanup()

	cl, err := c.FindResource("k8s_ingress.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeK8sIngress, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}
func TestK8sIngressSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, k8sIngressDisabled)
	defer cleanup()

	cl, err := c.FindResource("k8s_ingress.testing")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const k8sIngressDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

k8s_cluster "testing" {
	network {
		name = "network.test"
	}
	driver = "k3s"
}

k8s_ingress "testing" {
	cluster = "k8s_cluster.testing"
}
`
const k8sIngressDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

k8s_cluster "testing" {
	network {
		name = "network.test"
	}
	driver = "k3s"
}

k8s_ingress "testing" {
	disabled = true
	cluster = "k8s_cluster.testing"
}
`
