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

	"github.com/stretchr/testify/assert"
)

var lock = sync.Mutex{}

func setupTests(returnVals map[string]error) (*Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	log.SetOutput(ioutil.Discard)

	p := &[]*mocks.MockProvider{}

	cl := &Clients{}
	e := &Engine{
		clients:     cl,
		log:         hclog.NewNullLogger(),
		getProvider: generateProviderMock(p, returnVals),
	}

	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	return e, nil, p, func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
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
	return filepath.Join(path, "/functional_tests/test_fixtures", tests)
}

func TestApplyCallsProviderInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Apply("../../functional_tests/test_fixtures/single_k3s_cluster")
	assert.NoError(t, err)

	// should have called in order
	assert.Equal(t, "cloud", (*mp)[0].Config().Info().Name)
	assert.Equal(t, "k3s", (*mp)[1].Config().Info().Name)

	// due to paralel nature of the DAG, these two elements can appear in any order
	assert.Contains(t, []string{"consul-http", "consul"}, (*mp)[2].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul"}, (*mp)[3].Config().Info().Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Apply("../../functional_tests/test_fixtures/single_k3s_cluster")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 4)
}

func TestApplyCallsProviderCreateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	err := e.Apply("../../functional_tests/test_fixtures/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Destroy("../../functional_tests/test_fixtures/single_k3s_cluster", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 4)
}

func TestApplyCallsProviderDestroyInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Destroy("../../functional_tests/test_fixtures/single_k3s_cluster", true)
	assert.NoError(t, err)

	// due to paralel nature of the DAG, these two elements can appear in any order
	assert.Contains(t, []string{"consul-http", "consul"}, (*mp)[0].Config().Info().Name)
	assert.Contains(t, []string{"consul-http", "consul"}, (*mp)[1].Config().Info().Name)

	// should have called in order
	assert.Equal(t, "k3s", (*mp)[2].Config().Info().Name)
	assert.Equal(t, "cloud", (*mp)[3].Config().Info().Name)

}

/*
func TestApplyGeneratesState(t *testing.T) {
	e, _, _, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Apply("boom")
	assert.NoError(t, err)

	// state should be saved to a file in json format
	f, err := os.Open(utils.StatePath())
	assert.NoError(t, err)

	s := []map[string]interface{}{}
	jd := json.NewDecoder(f)
	jd.Decode(&s)

	assert.Len(t, s, 6)
}

func TestApplyWithExistingStateDoesNotRecreateItems(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	// generate some state, use the initial network
	// ep := []providers.ConfigWrapper{providers.ConfigWrapper{Type: "config.Network", Value: c.Networks[0]}}
	// testCreateStateFile(ep)

	err := e.Apply("boom")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 5)
}

*/
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
