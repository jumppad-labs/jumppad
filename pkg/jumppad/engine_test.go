package jumppad

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/constants"
	"github.com/jumppad-labs/jumppad/pkg/providers"
	"github.com/jumppad-labs/jumppad/pkg/providers/mocks"
	"github.com/jumppad-labs/jumppad/pkg/utils"

	"github.com/stretchr/testify/require"
	assert "github.com/stretchr/testify/require"
)

var lock = sync.Mutex{}

func setupTests(t *testing.T, returnVals map[string]error) (*EngineImpl, *[]*mocks.MockProvider) {
	return setupTestsBase(t, returnVals, "")
}

func setupTestsWithState(t *testing.T, returnVals map[string]error, state string) (*EngineImpl, *[]*mocks.MockProvider) {
	return setupTestsBase(t, returnVals, state)
}

func setupTestsBase(t *testing.T, returnVals map[string]error, state string) (*EngineImpl, *[]*mocks.MockProvider) {
	log.SetOutput(ioutil.Discard)

	p := &[]*mocks.MockProvider{}

	cl := &clients.Clients{}
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
	return func(c types.Resource, cc *clients.Clients) providers.Provider {
		lock.Lock()
		defer lock.Unlock()

		m := mocks.New(c)

		val := returnVals[c.Metadata().Name]
		m.On("Create").Return(val)
		m.On("Destroy").Return(val)
		m.On("Refresh").Return(val)

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

func testLoadState(t *testing.T, e *EngineImpl) *hclconfig.Config {
	c, err := resources.LoadState()
	require.NoError(t, err)

	return c
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
	require.NoError(t, err)

	require.Len(t, e.config.Resources, 5)

	// Because of the DAG both network and template are top level resources
	require.ElementsMatch(t,
		[]string{
			"default",
			"onprem",
			"default",
			"consul_config",
		},
		[]string{
			(*mp)[0].Config().Metadata().Name,
			(*mp)[1].Config().Metadata().Name,
			(*mp)[2].Config().Metadata().Name,
			(*mp)[3].Config().Metadata().Name,
		},
	)

	require.Equal(t, "consul", (*mp)[4].Config().Metadata().Name)
	require.Equal(t, "consul_addr", (*mp)[5].Config().Metadata().Name)
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
	require.Len(t, e.config.Resources, 5)

	// then the container should be created
	require.Equal(t, "consul", (*mp)[4].Config().Metadata().Name)

	// finally the provider for the image cache should be updated
	require.Equal(t, "consul_addr", (*mp)[5].Config().Metadata().Name)

	// check the variable has overridden the image
	cont := (*mp)[4].Config().(*resources.Container)
	require.Equal(t, "consul:1.8.1", cont.Image.Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply("../../examples/single_k3s_cluster")
	require.NoError(t, err)

	// should have call create for each resource in the config
	// and twice for ImageCache that is manually added.
	// the provider for image cache is called once for the initial creation
	// and once for each network
	rc := len(e.config.Resources) + 1
	testAssertMethodCalled(t, mp, "Create", rc)

	// the state should also contain 10 resources
	sf := testLoadState(t, e)
	require.Equal(t, 11, sf.ResourceCount())
}

func TestApplyDoesNotCallsProviderCreateWhenInState(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	_, err := e.Apply("../../examples/single_file")
	require.NoError(t, err)

	// should only be called for resources that are not in the state
	testAssertMethodCalled(t, mp, "Create", 4)

	// the state should also contain 7 resources
	sf := testLoadState(t, e)
	require.Equal(t, 7, sf.ResourceCount())
}

func TestApplyNotCallsProviderCreateForDisabledResources(t *testing.T) {
	e, mp := setupTests(t, nil)

	// contains 2 resources one is disabled
	_, err := e.Apply("../../examples/disabled")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 2) // ImageCache is always created

	// disabled resources should still be added to the state
	sf := testLoadState(t, e)

	// should contain 2 from the config plus the image cache
	require.Equal(t, 3, sf.ResourceCount())

	// the resource should be in the state but there should be no status
	r, err := sf.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.Nil(t, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplyShouldNotAddDuplicateDisabledResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	// contains 2 resources one is disabled
	_, err := e.Apply("../../examples/disabled")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created

	// disabled resources should still be added to the state
	sf := testLoadState(t, e)

	// should contain 2 from the config plus the image cache
	// should not duplicate the disabled container as this already exists in the
	// state
	require.Equal(t, 4, sf.ResourceCount())

	// the status should be set to disabled
	r, err := sf.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.Equal(t, constants.StatusDisabled, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplySetsCreatedStatusForEachResource(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Equal(t, 5, e.config.ResourceCount())

	// should only call create and destroy for the cache as this is pending update
	testAssertMethodCalled(t, mp, "Create", 6) // ImageCache is always created

	sf := testLoadState(t, e)

	r, err := sf.FindResource("resource.container.consul")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	r, err = sf.FindResource("resource.network.onprem")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	r, err = sf.FindResource("resource.template.consul_config")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTests(t, map[string]error{"onprem": fmt.Errorf("boom")})

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.Error(t, err)

	// should have call create for each provider
	// there are two top level config items, template and network
	// network will fail
	// ImageCache should always be created
	testAssertMethodCalled(t, mp, "Create", 3)

	sf := testLoadState(t, e)

	// should set failed status for network
	r, err := sf.FindResource("resource.network.onprem")
	require.NoError(t, err)
	require.Equal(t, constants.StatusFailed, r.Metadata().Properties[constants.PropertyStatus])

	// should set created status for template
	r, err = sf.FindResource("resource.template.consul_config")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	// container should not be in the state
	_, err = sf.FindResource("resource.container.consul")
	require.Error(t, err)
}

func TestApplyCallsProviderDestroyAndCreateForFailedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, failedState)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 6) // ImageCache is always created
}

func TestApplyCallsProviderDestroyForTaintedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, taintedState)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 6) // ImageCache is always created
}

func TestApplyCallsProviderDestroyForDisabledResources(t *testing.T) {
	// resources that were created but subsequently set to disabled should be
	// destroyed
	e, mp := setupTestsWithState(t, nil, disabledAndCreatedState)

	_, err := e.Apply(".")
	require.NoError(t, err)

	// get the disabled resource
	r, err := e.config.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.NotNil(t, r)

	// property should be disabled
	require.Equal(t, constants.StatusDisabled, r.Metadata().Properties[constants.PropertyStatus])

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1, r)
	testAssertMethodCalled(t, mp, "Create", 0) // ImageCache is always created
}

func TestApplyCallsProviderRefreshForCreatedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should only call one time as there is only one item in the state
	// that is in the config
	testAssertMethodCalled(t, mp, "Refresh", 1)
}

func TestApplyCallsProviderRefreshWithErrorHaltsExecution(t *testing.T) {
	e, mp := setupTestsWithState(t, map[string]error{"consul_config": fmt.Errorf("boom")}, existingState)

	_, err := e.Apply("../../examples/single_file/container.hcl")
	require.Error(t, err)

	// should only call one time as there is only one item in the state
	// that is in the config
	testAssertMethodCalled(t, mp, "Refresh", 1)
	testAssertMethodCalled(t, mp, "Create", 2)
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	err := e.Destroy()
	require.NoError(t, err)

	// should have call create for each provider
	// and once for the image cache
	testAssertMethodCalled(t, mp, "Destroy", 4)

	// state should be removed
	require.NoFileExists(t, utils.StatePath())
}

func TestDestroyNotCallsProviderDestroyForResourcesDisabled(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	err := e.Destroy()
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 2)
	testAssertMethodCalled(t, mp, "Create", 0) // ImageCache is always created

	// state should be removed
	require.NoFileExists(t, utils.StatePath())
}

func TestDestroyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTestsWithState(t, map[string]error{"mycontainer": fmt.Errorf("boom")}, complexState)

	err := e.Destroy()
	require.Error(t, err)

	// should have call destroy for each provider
	testAssertMethodCalled(t, mp, "Destroy", 2)

	// state should not be removed
	require.FileExists(t, utils.StatePath())
}

func TestDestroyFailSetsStatus(t *testing.T) {
	e, _ := setupTestsWithState(t, map[string]error{"mycontainer": fmt.Errorf("boom")}, complexState)

	err := e.Destroy()
	require.Error(t, err)

	r, _ := e.config.FindResource("resource.container.mycontainer")
	require.Equal(t, constants.StatusFailed, r.Metadata().Properties[constants.PropertyStatus])
}

func TestParseConfig(t *testing.T) {
	e, mp := setupTests(t, nil)

	r, err := e.ParseConfig("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, r, 4)

	// should not have crated any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)
}

func TestParseWithVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	r, err := e.ParseConfigWithVariables("../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	require.NoError(t, err)

	require.Len(t, r, 4)

	// should not have crated any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)

	c, err := e.config.FindResource("resource.container.consul")
	require.NoError(t, err)
	require.Equal(t, "consul:1.8.1", c.(*resources.Container).Image.Name)
}

func testAssertMethodCalled(t *testing.T, p *[]*mocks.MockProvider, method string, n int, resource ...types.Resource) {
	callCount := 0

	calls := map[string][]string{}

	for _, pm := range *p {
		// are we trying to filter on a specific resource
		if len(resource) == 1 {
			if resource[0] != pm.Config() {
				continue
			}
		}

		for _, c := range pm.Calls {
			calls[pm.Config().Metadata().Name] = append(calls[pm.Config().Metadata().Name], c.Method)

			if c.Method == method {
				callCount++
			}
		}
	}

	callStrings := []string{}
	for k, v := range calls {
		calls := strings.Join(v, ",")
		callStrings = append(callStrings, fmt.Sprintf("%s[%s]", k, calls))
	}

	callString := strings.Join(callStrings, " ")

	require.Equal(t, n, callCount, fmt.Sprintf("expected %d calls, actual calls %d: %s", n, callCount, callString))
}

var failedState = `
{
  "resources": [
	{
      "name": "onprem",
 	    "properties": {
				"status": "failed"
			},
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var taintedState = `
{
  "resources": [
	{
      "name": "onprem",
 	    "properties": {
				"status": "tainted"
			},
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var existingState = `
{
  "resources": [
	{
      "name": "cloud",
      "properties": {
				"status": "created"
			},
      "subnet": "10.15.0.0/16",
      "type": "network"
	},
	{
      "name": "default",
      "properties": {
				"status": "created"
			},
      "type": "image_cache"
	},
	{
      "name": "container",
      "properties": {
				"status": "created"
			},
      "type": "container"
	},
	{
      "name": "consul_config",
      "properties": {
				"status": "created"
			},
      "type": "template"
	}
  ]
}
`

var complexState = `
{
  "resources": [
	{
      "name": "cloud",
      "subnet": "10.15.0.0/16",
      "properties": {
				"status": "created"
			},
      "type": "network"
	},
	{
      "name": "default",
      "type": "image_cache",
      "properties": {
				"status": "created"
			},
			"depends_on": ["resource.network.cloud"]
	},
	{
      "name": "mytemplate",
      "properties": {
				"status": "created"
			},
      "type": "template"
	},
	{
      "name": "mycontainer",
      "type": "container",
      "properties": {
				"status": "created"
			},
			"depends_on": ["resource.network.cloud", "resource.template.mytemplate"]
	}
  ]
}
`

var disabledState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "default",
      "status": "created",
      "type": "image_cache"
	},
	{
      "name": "dc1_enabled",
      "properties": {
				"status": "created"
			},
      "subnet": "10.15.0.0/16",
      "type": "network"
	},
	{
		 "disabled": true,
      "name": "consul_disabled",
      "properties": {
				"status": "disabled"
			},
      "type": "container"
	}
  ]
}
`

var disabledAndCreatedState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "default",
      "properties": {
				"status": "created"
			},
      "type": "image_cache"
	},
	{
      "name": "dc1_enabled",
      "properties": {
				"status": "created"
			},
      "subnet": "10.15.0.0/16",
      "type": "network"
	},
	{
		 "disabled": true,
      "name": "consul_disabled",
      "properties": {
				"status": "created"
			},
      "type": "container"
	}
  ]
}
`
