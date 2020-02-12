package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testSetupConfig() *Config {
	c := New()
	c.Resources = []Resource{NewK8sCluster("test")}

	return c
}

func TestResourceCount(t *testing.T) {

	//assert.Equal(t, 10, c.ResourceCount())
}

func TestFindResourceFindsCluster(t *testing.T) {
	c := testSetupConfig()

	cl, err := c.FindResource("k8s_cluster.test")
	assert.NoError(t, err)
	assert.Equal(t, c.Resources[0], cl)
}

func TestFindResourceReturnsNotFoundError(t *testing.T) {
	c := testSetupConfig()

	cl, err := c.FindResource("cluster.notexist")
	assert.Error(t, err)
	assert.IsType(t, err, ResourceNotFoundError{})
	assert.Equal(t, c.Resources[0], cl)
}

func TestAddResourceAddsAResouce(t *testing.T) {
	c := testSetupConfig()

	cl := NewK8sCluster("mikey")
	err := c.AddResource(cl)
	assert.NoError(t, err)

	cl2, err := c.FindResource("k8s_cluster.mikey")
	assert.NoError(t, err)
	assert.Equal(t, cl, cl2)
}
