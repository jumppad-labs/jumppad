package shipyard

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/providers/mocks"
	"github.com/shipyard-run/shipyard/pkg/shipyard/constants"
	"github.com/shipyard-run/shipyard/pkg/utils"

	"github.com/stretchr/testify/require"
	assert "github.com/stretchr/testify/require"
)

var lock = sync.Mutex{}

func setupTests(t *testing.T, returnVals map[string]error) (Engine, *[]*mocks.MockProvider) {
	return setupTestsBase(t, returnVals, "")
}

func setupTestsWithState(t *testing.T, returnVals map[string]error, state string) (Engine, *[]*mocks.MockProvider) {
	return setupTestsBase(t, returnVals, state)
}

func setupTestsBase(t *testing.T, returnVals map[string]error, state string) (Engine, *[]*mocks.MockProvider) {
	log.SetOutput(ioutil.Discard)

	p := &[]*mocks.MockProvider{}

	cl := &Clients{}
	e := &EngineImpl{
		clients:     cl,
		log:         hclog.NewNullLogger(),
		getProvider: generateProviderMock(p, returnVals),
	}

	setupState(t, state)

	return e, p
}

func setupState(t *testing.T, state string) {
	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), dir)

	// write the state file
	if state != "" {
		os.MkdirAll(utils.StateDir(), os.ModePerm)
		f, err := os.Create(utils.StatePath())
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.WriteString(state)
		if err != nil {
			panic(err)
		}
	}

	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), home)
		os.RemoveAll(dir)
	})
}

func generateProviderMock(mp *[]*mocks.MockProvider, returnVals map[string]error) getProviderFunc {
	return func(c types.Resource, cc *Clients) providers.Provider {
		fmt.Println("create provider for", c.Metadata().Name)

		lock.Lock()
		defer lock.Unlock()

		m := mocks.New(c)

		val := returnVals[c.Metadata().Name]
		m.On("Create").Return(val)
		m.On("Destroy").Return(val)

		*mp = append(*mp, m)
		return m
	}
}

func getTestFiles(tests string) string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path := path.Dir(e)
	return filepath.Join(path, "/examples", tests)
}

func TestNewCreatesClients(t *testing.T) {
	e, err := New(hclog.NewNullLogger())
	assert.NoError(t, err)

	cl := e.GetClients()

	assert.NotNil(t, cl.Kubernetes)
	assert.NotNil(t, cl.Helm)
	assert.NotNil(t, cl.Command)
	assert.NotNil(t, cl.HTTP)
	assert.NotNil(t, cl.Nomad)
	assert.NotNil(t, cl.Getter)
	assert.NotNil(t, cl.Browser)
	assert.NotNil(t, cl.ImageLog)
	assert.NotNil(t, cl.Connector)
}

func TestApplyWithSingleFile(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	assert.NoError(t, err)

	// Because of the DAG both network and template are top level resources
	assert.ElementsMatch(t,
		[]string{
			"onprem",
			"consul_config",
		},
		[]string{
			(*mp)[0].Config().Metadata().Name,
			(*mp)[1].Config().Metadata().Name,
		},
	)

	assert.Equal(t, "consul", (*mp)[2].Config().Metadata().Name)
}

func TestApplyAddsImageCache(t *testing.T) {
	e, _ := setupTests(t, nil)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	dc := e.ResourceCountForType(resources.TypeImageCache)
	require.Equal(t, 1, dc)
}

func TestApplyWithSingleFileAndVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.ApplyWithVariables("../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	require.NoError(t, err)

	// network should have been created with the name 'onprem'
	// template should also be created,
	// and also the image cache, all of these are top level items
	require.Contains(t, []string{"onprem", "consul_config"}, (*mp)[0].Config().Metadata().Name)
	require.Contains(t, []string{"onprem", "consul_config"}, (*mp)[1].Config().Metadata().Name)

	// then the container should be created
	require.Equal(t, "consul", (*mp)[2].Config().Metadata().Name)

	// finally the provider for the image cache should be updated
	require.Equal(t, "default", (*mp)[3].Config().Metadata().Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 9)
	//assert.Len(t, res, 4)
}

func TestApplyNotCallsProviderDestroyAndCreateForResourcesDisabled(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 0)
	testAssertMethodCalled(t, mp, "Create", 2) // ImageCache is always created

	// check state of disabled remains disabled
	d, err := os.ReadFile(utils.StatePath())
	require.NoError(t, err)

	parser := setupHCLConfig(nil, nil, nil)
	c, err := parser.UnmarshalJSON(d)
	require.NoError(t, err)

	r, err := c.FindResource("container.dc1")
	assert.NoError(t, err)
	assert.True(t, r.Metadata().Disabled)
}

func TestApplyCallsProviderDestroyAndCreateForResourcesFailed(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, failedState)

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 2) // ImageCache is always created
}

func TestApplyCallsProviderDestroyForResourcesPendingModification(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, modificationState)

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 2) // ImageCache is always created
}

func TestApplyReturnsErrorWhenProviderDestroyForResourcesPendingorFailed(t *testing.T) {
	e, mp := setupTestsWithState(t, map[string]error{"dc1": fmt.Errorf("boom")}, failedState)

	_, err := e.Apply("")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created
}

func TestApplyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTests(t, map[string]error{"cloud": fmt.Errorf("boom")})

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 2)
}

func TestApplySetsStatusForEachResource(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, mergedState)

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should only call create and destroy for the cache as this is pending update
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created
}

func TestParseConfig(t *testing.T) {
	e, mp := setupTests(t, nil)

	err := e.ParseConfig("../../examples/single_file/container.hcl")
	assert.NoError(t, err)

	assert.Equal(t, 3, e.(*EngineImpl).config.ResourceCount())

	// should not have crated any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)
}

func TestParseWithVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	err := e.ParseConfigWithVariables("../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	assert.NoError(t, err)

	assert.Equal(t, 3, e.(*EngineImpl).config.ResourceCount())

	// should not have crated any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, mp := setupTests(t, nil)

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 9)
}

func TestDestroyNotCallsProviderDestroyForResourcesDisabled(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	err := e.Destroy("", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 2)
	testAssertMethodCalled(t, mp, "Create", 0) // ImageCache is always created

	// check state of disabled remains disabled
	d, err := os.ReadFile(utils.StatePath())
	require.NoError(t, err)

	parser := setupHCLConfig(nil, nil, nil)
	c, err := parser.UnmarshalJSON(d)
	require.NoError(t, err)

	_, err = c.FindResource("container.dc1")
	assert.Error(t, err) // resource should not exist
}

func TestDestroyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTests(t, map[string]error{"k3s": fmt.Errorf("boom")})

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 7)
}

func TestDestroyFailSetsStatus(t *testing.T) {
	e, mp := setupTests(t, map[string]error{"cloud": fmt.Errorf("boom")})

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 9)
	assert.Equal(t, constants.StatusFailed, (*mp)[8].Config().Metadata().Properties[constants.PropertyStatus])
}

func TestDestroyCallsProviderDestroyInCorrectOrder(t *testing.T) {
	e, mp := setupTests(t, nil)

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.NoError(t, err)

	// due to paralel nature of the DAG, these elements can appear in any order
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[0].Config().Metadata().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[1].Config().Metadata().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[2].Config().Metadata().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[3].Config().Metadata().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[4].Config().Metadata().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault", "connector", "consul-lan", "KUBECONFIG"}, (*mp)[5].Config().Metadata().Name)

	assert.Contains(t, []string{"k3s"}, (*mp)[6].Config().Metadata().Name)
	assert.Contains(t, []string{"docker-cache"}, (*mp)[7].Config().Metadata().Name)
	assert.Contains(t, []string{"cloud"}, (*mp)[8].Config().Metadata().Name)
}

func testAssertMethodCalled(t *testing.T, p *[]*mocks.MockProvider, method string, n int, args ...interface{}) {
	callCount := 0

	for _, pm := range *p {
		// cast the provider into a mock
		for _, c := range pm.Calls {
			if c.Method == method {
				callCount++
			}
		}
	}

	if callCount != n {
		t.Fatalf("Expected %d calls got %d", n, callCount)
	}
}

var failedState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "failed",
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var modificationState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "pending_modification",
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var mergedState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "pending_update",
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var disabledState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "pending_creation",
      "subnet": "10.15.0.0/16",
      "type": "network"
	},
	{
      "name": "dc1",
      "status": "disabled",
      "type": "container"
	}
  ]
}
`
