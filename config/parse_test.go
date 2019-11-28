package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleKubernetesCluster(t *testing.T) {
	c, err := ParseFolder("./examples/single-cluster-k8s")

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// validate clusters
	assert.Len(t, c.Clusters, 1)

	c1 := c.Clusters[0]
	assert.Equal(t, "default", c1.name)
	assert.Equal(t, "1.16.0", c1.Version)
	assert.Equal(t, 3, c1.Nodes)
	assert.Equal(t, "network.k8s", c1.Network)

	// validate networks
	assert.Len(t, c.Networks, 1)

	n1 := c.Networks[0]
	assert.Equal(t, "k8s", n1.name)
	assert.Equal(t, "10.4.0.0/16", n1.Subnet)

	// validate references
	err = ParseReferences(c)
	assert.NoError(t, err)

	assert.Equal(t, n1, c1.networkRef)
}
