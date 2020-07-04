// +build !race

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
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/providers/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"

	"github.com/stretchr/testify/assert"
)

var lock = sync.Mutex{}

func setupTests(returnVals map[string]error) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	return setupTestsBase(returnVals, "")
}

func setupTestsWithState(returnVals map[string]error, state string) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	return setupTestsBase(returnVals, state)
}

func setupState(state string) func() {
	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

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

	return func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
}

func setupTestsBase(returnVals map[string]error, state string) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	log.SetOutput(ioutil.Discard)

	p := &[]*mocks.MockProvider{}

	cl := &Clients{}
	e := &EngineImpl{
		clients:     cl,
		log:         hclog.NewNullLogger(),
		getProvider: generateProviderMock(p, returnVals),
	}

	return e, nil, p, setupState(state)
}

func generateProviderMock(mp *[]*mocks.MockProvider, returnVals map[string]error) getProviderFunc {
	return func(c config.Resource, cc *Clients) providers.Provider {
		lock.Lock()
		defer lock.Unlock()

		m := mocks.New(c)

		val := returnVals[c.Info().Name]
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

func TestApplyCallsProviderInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.NoError(t, err)

	// should have called in order
	assert.Equal(t, "cloud", (*mp)[0].Config().Info().Name)
	assert.Equal(t, "k3s", (*mp)[1].Config().Info().Name)

	// due to paralel nature of the DAG, these two elements can appear in any order
	assert.Contains(t, []string{"consul-http", "consul", "vault", "vault-http"}, (*mp)[2].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault", "vault-http"}, (*mp)[3].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault", "vault-http"}, (*mp)[4].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault", "vault-http"}, (*mp)[5].Config().Info().Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 6)
	//assert.Len(t, res, 4)
}

func TestApplyCallsProviderDestroyForResourcesPendingorFailed(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(nil, failedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplyReturnsErrorWhenProviderDestroyForResourcesPendingorFailed(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(map[string]error{"dc1": fmt.Errorf("boom")}, failedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 0)
}

func TestApplyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplyCallsProviderCreateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplySetsStatusForEachResource(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(nil, mergedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should not call create as this is pending update
	testAssertMethodCalled(t, mp, "Create", 0)
	assert.Len(t, *mp, 0)
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 6)
}

func TestDestroyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"k3s": fmt.Errorf("boom")})
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 5)
}

func TestDestroyFailSetsStatus(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 6)
	assert.Equal(t, config.Failed, (*mp)[5].Config().Info().Status)
}

func TestDestroyCallsProviderDestroyInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.NoError(t, err)

	// due to paralel nature of the DAG, these two elements can appear in any order
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault"}, (*mp)[0].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault"}, (*mp)[1].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault"}, (*mp)[2].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul", "vault-http", "vault"}, (*mp)[3].Config().Info().Name)

	// should have called in order
	assert.Equal(t, "k3s", (*mp)[4].Config().Info().Name)
	assert.Equal(t, "cloud", (*mp)[5].Config().Info().Name)

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
