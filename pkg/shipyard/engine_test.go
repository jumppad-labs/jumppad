package shipyard

func testCreateStateFile(p []providers.ConfigWrapper) {
	e := Engine{log: hclog.NewNullLogger()}
	e.state = p
	e.saveState()
}
import (
	"encoding/json"
	"os"
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

func setupTests() (*Engine, *config.Config, func()) {
	//md := &clients.MockDocker{}
	// n1 := config.NewNetwork("network1")
	c1 := config.NewContainer("container1")
	c1.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "network.network1"}}
	cl1 := config.NewK8sCluster("cluster1")
	cl1.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "network.network1"}}
	h1 := config.NewHelm("helm1")
	h1.Cluster = "k8s_cluster.cluster1"
	i1 := config.NewIngress("ingress1")
	i1.Target = "cluster.cluster1"

	c := config.New()
	// c.Containers = []*config.Container{c1}
	// c.Clusters = []*config.Cluster{cl1}
	// c.Networks = []*config.Network{n1}
	// c.Ingresses = []*config.Ingress{i1}
	// c.HelmCharts = []*config.Helm{h1}

	cl := &Clients{}
	e := &Engine{
		clients:           cl,
		config:            c,
		log:               hclog.NewNullLogger(),
		generateProviders: generateProvidersMock,
		stateLock:         sync.Mutex{},
	}

	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	return e, c, func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, _, cleanup := setupTests()
	defer cleanup()

	err := e.Apply("boom")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, e.providers, "Create", 6)
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, _, cleanup := setupTests()
	defer cleanup()

	err := e.Apply("boom")
	assert.NoError(t, err)

	err = e.Destroy("boom", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, e.providers, "Destroy", 6)
}

func TestApplyGeneratesState(t *testing.T) {
	e, _, cleanup := setupTests()
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
	e, _, cleanup := setupTests()
	defer cleanup()

	// generate some state, use the initial network
	// ep := []providers.ConfigWrapper{providers.ConfigWrapper{Type: "config.Network", Value: c.Networks[0]}}
	// testCreateStateFile(ep)

	err := e.Apply("boom")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, e.providers, "Create", 5)
}

func testCreateStateFile(p []providers.ConfigWrapper) {
	e := Engine{log: hclog.NewNullLogger()}
	e.state = p
	e.saveState()
}

func testAssertMethodCalled(t *testing.T, p [][]providers.Provider, method string, n int, args ...interface{}) {
	callCount := 0

	for _, pg := range p {
		for _, p := range pg {
			// cast the provider into a mock
			pm := p.(*mocks.MockProvider)
			for _, c := range pm.Calls {
				if c.Method == method {
					callCount++
				}
			}
		}
	}

	if callCount != n {
		t.Fatalf("Expected %d calls got %d", n, callCount)
	}
}

// generate mock providers rathen than concrete implemetations
func generateProvidersMock(c *config.Config, cc *Clients, l hclog.Logger) [][]providers.Provider {
	oc := make([][]providers.Provider, 7)
	oc[0] = make([]providers.Provider, 0)
	oc[1] = make([]providers.Provider, 0)
	oc[2] = make([]providers.Provider, 0)
	oc[3] = make([]providers.Provider, 0)
	oc[4] = make([]providers.Provider, 0)
	oc[5] = make([]providers.Provider, 0)
	oc[6] = make([]providers.Provider, 0)

	// add the wan
	// cw := providers.ConfigWrapper{Type: "config.Network", Value: c.WAN}
	// p := mocks.New(cw)
	// p.On("Create").Return(nil)
	// p.On("Destroy").Return(nil)
	// p.On("Config").Return(cw)

	// oc[0] = append(oc[0], p)

	// for _, n := range c.Networks {
	// 	cw = providers.ConfigWrapper{Type: "config.Network", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[0] = append(oc[0], p)
	// }

	// for _, n := range c.Containers {
	// 	cw = providers.ConfigWrapper{Type: "config.Container", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[1] = append(oc[1], p)
	// }

	// for _, n := range c.Ingresses {
	// 	cw = providers.ConfigWrapper{Type: "config.Ingress", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[1] = append(oc[1], p)
	// }

	// if c.Docs != nil {
	// 	cw = providers.ConfigWrapper{Type: "config.Docs", Value: c.Docs}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[1] = append(oc[1], p)
	// }

	// for _, n := range c.Clusters {
	// 	cw = providers.ConfigWrapper{Type: "config.Cluster", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[2] = append(oc[2], p)
	// }

	// for _, n := range c.HelmCharts {
	// 	cw = providers.ConfigWrapper{Type: "config.Helm", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[3] = append(oc[3], p)
	// }

	// for _, n := range c.K8sConfig {
	// 	cw = providers.ConfigWrapper{Type: "config.K8sConfig", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[4] = append(oc[4], p)
	// }

	// for _, n := range c.LocalExecs {
	// 	cw = providers.ConfigWrapper{Type: "config.LocalExec", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[6] = append(oc[6], p)
	// }

	// for _, n := range c.RemoteExecs {
	// 	cw = providers.ConfigWrapper{Type: "config.RemoteExec", Value: n}
	// 	p = mocks.New(cw)
	// 	p.On("Create").Return(nil)
	// 	p.On("Destroy").Return(nil)
	// 	p.On("Config").Return(cw)

	// 	oc[6] = append(oc[6], p)
	// }

	return oc
}
