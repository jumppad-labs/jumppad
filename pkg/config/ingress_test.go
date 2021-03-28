package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesIngress(t *testing.T) {
	c := NewIngress("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeIngress, c.Type)
}
func TestIngressCreatesCorrectly(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, ingressDefault)
	defer cleanup()

	cl, err := c.FindResource("ingress.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeIngress, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}
func TestIngressSetsDisabled(t *testing.T) {
	c, _, cleanup := setupTestConfig(t, ingressDisabled)
	defer cleanup()

	cl, err := c.FindResource("ingress.testing")
	assert.NoError(t, err)

	assert.Equal(t, Disabled, cl.Info().Status)
}

const ingressDefault = `
network "test" {
	subnet = "10.0.0.0/24"
}

k8s_cluster "testing" {
	network {
		name = "network.test"
	}
	driver = "k3s"
}

ingress "testing" {
	destination {
		driver = "k8s"
		config {
			port = 8080
		}
	}
	
	source {
		driver = "k8s"
		config {
			port = 8080
		}
	}
}
`
const ingressDisabled = `
network "test" {
	subnet = "10.0.0.0/24"
}

k8s_cluster "testing" {

	network {
		name = "network.test"
	}
	driver = "k3s"
}

ingress "testing" {
	disabled = true

	destination {
		driver = "k8s"
		config {
			port = 8080
		}
	}
	
	source {
		driver = "k8s"
		config {
			port = 8080
		}
	}
}
`
