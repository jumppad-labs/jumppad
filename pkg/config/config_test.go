package config

import (
	"fmt"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func testSetupConfig(t *testing.T) *Config {
	cache := NewImageCache("docker-cache")
	net1 := NewNetwork("cloud")
	cl1 := NewK8sCluster("test.dev")
	cl1.DependsOn = []string{"network.cloud"}
	cache.DependsOn = []string{"network.cloud"}

	cl2 := NewK8sCluster("test.dev")
	cl2.Module = "sub_module"

	c := New()
	c.AddResource(net1)
	c.AddResource(cl1)
	c.AddResource(cache)
	c.AddResource(cl2)

	return c
}

func testSetupModuleConfig(t *testing.T) *Config {
	net1 := NewNetwork("cloud")
	net1.Module = "test"

	cl1 := NewK8sCluster("test.dev")
	cl1.DependsOn = []string{"module.test"}

	c := New()
	err := c.AddResource(net1)
	assert.NoError(t, err)

	err = c.AddResource(cl1)
	assert.NoError(t, err)

	return c
}

func TestResourceCount(t *testing.T) {

	//assert.Equal(t, 10, c.ResourceCount())
}

func TestResourceAddChildSetsDetails(t *testing.T) {
	c := testSetupConfig(t)
	cl := NewK8sCluster("newtest")

	c.Resources[0].AddChild(cl)

	assert.Equal(t, c.Resources[0].Info().Config, cl.Info().Config)
	assert.Equal(t, c.Resources[0].Info().Type, cl.Type)
}

func TestFindResourceFindsCluster(t *testing.T) {
	c := testSetupConfig(t)

	cl, err := c.FindResource("k8s_cluster.test.dev")
	assert.NoError(t, err)
	assert.Equal(t, c.Resources[1], cl)
}

func TestFindResourceFindsClusterInModule(t *testing.T) {
	c := testSetupConfig(t)

	cl, err := c.FindResource("sub_module.k8s_cluster.test.dev")
	assert.NoError(t, err)
	assert.Equal(t, c.Resources[3], cl)
}

func TestFindResourceReturnsNotFoundError(t *testing.T) {
	c := testSetupConfig(t)

	cl, err := c.FindResource("container.notexist")
	assert.Error(t, err)
	fmt.Println(err)
	assert.IsType(t, ResourceNotFoundError{}, err)
	assert.Nil(t, cl)
}

func TestFindDependentResourceFindsResource(t *testing.T) {
	c := testSetupConfig(t)

	r, err := c.Resources[0].FindDependentResource("k8s_cluster.test.dev")
	assert.NoError(t, err)
	assert.Equal(t, c.Resources[1], r)
}

func TestAddResourceAddsAResouce(t *testing.T) {
	c := testSetupConfig(t)

	cl := NewK8sCluster("mikey")
	err := c.AddResource(cl)
	assert.NoError(t, err)

	cl2, err := c.FindResource("k8s_cluster.mikey")
	assert.NoError(t, err)
	assert.Equal(t, cl, cl2)
}

func TestAddResourceExistsReturnsError(t *testing.T) {
	c := testSetupConfig(t)

	err := c.AddResource(c.Resources[3])
	assert.Error(t, err)
}

func TestAddResourceDifferentModuleSameNameOK(t *testing.T) {
	c := testSetupConfig(t)

	cl1 := NewK8sCluster("test.dev")
	cl1.Info().Module = "mymodule"

	err := c.AddResource(cl1)
	assert.NoError(t, err)
}

func TestRemoveResourceRemoves(t *testing.T) {
	c := testSetupConfig(t)

	err := c.RemoveResource(c.Resources[0])
	assert.NoError(t, err)
	assert.Len(t, c.Resources, 3)
}

func TestRemoveResourceNotFoundReturnsError(t *testing.T) {
	c := testSetupConfig(t)

	err := c.RemoveResource(nil)
	assert.Error(t, err)
	assert.Len(t, c.Resources, 4)
}

func TestDoYaLikeDAGGeneratesAGraph(t *testing.T) {
	c := testSetupConfig(t)

	d, err := c.DoYaLikeDAGs()
	assert.NoError(t, err)

	// check that all resources are added and dependencies created
	assert.Len(t, d.Edges(), 4)
}

func TestDoYaLikeDAGAddsDependencies(t *testing.T) {
	c := testSetupConfig(t)

	g, err := c.DoYaLikeDAGs()
	assert.NoError(t, err)

	// check the dependency tree of a cluster
	network, _ := c.FindResource("image_cache.docker-cache")
	s, err := g.Descendents(network)
	assert.NoError(t, err)

	// check that the network and a blueprint is returned
	list := s.List()
	assert.Contains(t, list, c.Resources[0])
	assert.Contains(t, list, &Blueprint{})
}

func TestDoYaLikeDAGAddsDependenciesForModules(t *testing.T) {
	c := testSetupModuleConfig(t)

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
	c := testSetupConfig(t)

	con := NewContainer("test")
	con.DependsOn = []string{"doesnot.exist"}

	c.AddResource(con)

	_, err := c.DoYaLikeDAGs()
	assert.Error(t, err)
}
