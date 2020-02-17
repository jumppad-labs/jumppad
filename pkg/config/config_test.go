package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testSetupConfig() *Config {
	c := New()
	c.AddResource(NewK8sCluster("test"))
	c.AddResource(NewK8sCluster("test2"))

	return c
}

func TestResourceCount(t *testing.T) {

	//assert.Equal(t, 10, c.ResourceCount())
}

func TestResourceAddChildSetsDetails(t *testing.T) {
	c := testSetupConfig()
	cl := NewK8sCluster("newtest")

	c.Resources[0].AddChild(cl)

	assert.Equal(t, c.Resources[0].Info().Config, cl.Info().Config)
	assert.Equal(t, c.Resources[0].Info().Type, cl.Type)
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
	assert.Nil(t, cl)
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

func TestAddResourceExistsReturnsError(t *testing.T) {
	c := testSetupConfig()

	err := c.AddResource(c.Resources[0])
	assert.Error(t, err)
}

func TestRemoveResourceRemoves(t *testing.T) {
	c := testSetupConfig()

	err := c.RemoveResource(c.Resources[0])
	assert.NoError(t, err)
	assert.Len(t, c.Resources, 1)
}

func TestRemoveResourceNotFoundReturnsError(t *testing.T) {
	c := testSetupConfig()

	err := c.RemoveResource(nil)
	assert.Error(t, err)
	assert.Len(t, c.Resources, 2)
}
