package shipyard

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/providers/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func setupTests() (*Engine, *config.Config, *mocks.MockProvider, func()) {
	//md := &clients.MockDocker{}
	w1 := &config.Network{Name: "wan"}
	n1 := &config.Network{Name: "network1"}
	c1 := &config.Container{Name: "container1", Network: "network.network1", NetworkRef: n1}
	cl1 := &config.Cluster{Name: "cluster1", Network: "network.network1", NetworkRef: n1}
	h1 := &config.Helm{Name: "helm1", Cluster: "cluster.cluster1", ClusterRef: cl1}
	i1 := &config.Ingress{Name: "ingress1", Target: "cluster.cluster1", TargetRef: cl1}

	c, _ := config.New()
	c.Containers = []*config.Container{c1}
	c.Clusters = []*config.Cluster{cl1}
	c.Networks = []*config.Network{n1}
	c.Ingresses = []*config.Ingress{i1}
	c.HelmCharts = []*config.Helm{h1}

	mp := &mocks.MockProvider{}
	mp.On("Create").Return(nil)
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Network", Value: w1}).Once()
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Network", Value: n1}).Once()
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Container", Value: c1}).Once()
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Cluster", Value: cl1}).Once()
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Helm", Value: h1}).Once()
	mp.On("Config").Return(providers.ConfigWrapper{Type: "config.Ingress", Value: i1}).Once()

	cl := &Clients{}
	e := New(c, cl, hclog.NewNullLogger())
	e.providers = generateProvidersMock(c, cl, mp, hclog.NewNullLogger())

	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	return e, c, mp, func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
}

func TestCorrectlyGeneratesProviders(t *testing.T) {
	_, c, _, cleanup := setupTests()
	defer cleanup()

	cl := &Clients{}

	// process the config
	oc := generateProvidersImpl(c, cl, hclog.NewNullLogger())

	// first element should be a network
	assert.Len(t, oc, 7)

	// WAN network
	_, ok := oc[0][0].(*providers.Network)
	assert.True(t, ok)

	_, ok = oc[0][1].(*providers.Network)
	assert.True(t, ok)

	_, ok = oc[1][0].(*providers.Container)
	assert.True(t, ok)

	_, ok = oc[1][1].(*providers.Ingress)
	assert.True(t, ok)

	_, ok = oc[2][0].(*providers.Cluster)
	assert.True(t, ok)

	_, ok = oc[3][0].(*providers.Helm)
	assert.True(t, ok)
}

func TestApplyCallsProviderCreate(t *testing.T) {
	e, _, mp, cleanup := setupTests()
	defer cleanup()

	err := e.Apply()
	assert.NoError(t, err)

	// should have call create for each provider
	mp.AssertNumberOfCalls(t, "Create", 6)
}

func TestApplyGeneratesState(t *testing.T) {
	e, _, _, cleanup := setupTests()
	defer cleanup()

	err := e.Apply()
	assert.NoError(t, err)

	// state should be saved to a file in json format
	f, err := os.Open(utils.StatePath())
	assert.NoError(t, err)

	s := []map[string]interface{}{}
	jd := json.NewDecoder(f)
	jd.Decode(&s)

	assert.Len(t, s, 6)
}

func TestNewFromStateCreatesCorrectly(t *testing.T) {
	e, _, _, cleanup := setupTests()
	defer cleanup()

	err := e.Apply()
	assert.NoError(t, err)

	// load from the state
	e, err = NewFromState(utils.StatePath(), hclog.NewNullLogger())
	assert.NoError(t, err)
}

// generate mock providers rathen than concrete implemetations
func generateProvidersMock(c *config.Config, cc *Clients, p providers.Provider, l hclog.Logger) [][]providers.Provider {
	oc := make([][]providers.Provider, 7)
	oc[0] = make([]providers.Provider, 0)
	oc[1] = make([]providers.Provider, 0)
	oc[2] = make([]providers.Provider, 0)
	oc[3] = make([]providers.Provider, 0)
	oc[4] = make([]providers.Provider, 0)
	oc[5] = make([]providers.Provider, 0)
	oc[6] = make([]providers.Provider, 0)

	// add the wan
	oc[0] = append(oc[0], p)

	for _, _ = range c.Networks {
		oc[0] = append(oc[0], p)
	}

	for _, _ = range c.Containers {
		oc[1] = append(oc[1], p)
	}

	for _, _ = range c.Ingresses {
		oc[1] = append(oc[1], p)
	}

	if c.Docs != nil {
		oc[1] = append(oc[1], p)
	}

	for _, _ = range c.Clusters {
		oc[2] = append(oc[2], p)
	}

	for _, _ = range c.HelmCharts {
		oc[3] = append(oc[3], p)
	}

	for _, _ = range c.K8sConfig {
		oc[4] = append(oc[4], p)
	}

	for _, _ = range c.LocalExecs {
		oc[6] = append(oc[6], p)
	}

	for _, _ = range c.RemoteExecs {
		oc[6] = append(oc[6], p)
	}

	return oc
}
