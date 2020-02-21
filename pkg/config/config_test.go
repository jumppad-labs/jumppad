package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testSetupConfig() *Config {
	net1 := NewNetwork("cloud")
	cl1 := NewK8sCluster("test")
	cl1.DependsOn = []string{"network.cloud"}

	c := New()
	c.AddResource(net1)
	c.AddResource(cl1)

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
	assert.Equal(t, c.Resources[1], cl)
}

func TestFindResourceReturnsNotFoundError(t *testing.T) {
	c := testSetupConfig()

	cl, err := c.FindResource("cluster.notexist")
	assert.Error(t, err)
	assert.IsType(t, err, ResourceNotFoundError{})
	assert.Nil(t, cl)
}

func TestFindDependentResourceFindsResource(t *testing.T) {
	c := testSetupConfig()

	r, err := c.Resources[0].FindDependentResource("k8s_cluster.test")
	assert.NoError(t, err)
	assert.Equal(t, c.Resources[1], r)
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

func TestDoYaLikeDAGGeneratesAGraph(t *testing.T) {
	c := testSetupConfig()

	d, err := c.DoYaLikeDAGs()
	assert.NoError(t, err)

	// check that all resources are added and dependencies created
	assert.Len(t, d.Edges(), 2)
}

func TestDoYaLikeDAGAddsDependencies(t *testing.T) {
	c := testSetupConfig()

	g, err := c.DoYaLikeDAGs()
	assert.NoError(t, err)

	// check the dependency tree of a cluster
	s, err := g.Descendents(c.Resources[1])
	assert.NoError(t, err)

	// check that the network and a blueprint is returned
	list := s.List()
	assert.Contains(t, list, c.Resources[0])
	assert.Contains(t, list, &Blueprint{})
}

func TestDoYaLikeDAGWithUnresolvedDependencyReturnsError(t *testing.T) {
	c := testSetupConfig()

	con := NewContainer("test")
	con.DependsOn = []string{"doesnot.exist"}

	c.AddResource(con)

	_, err := c.DoYaLikeDAGs()
	assert.Error(t, err)
}
