package shipyard

import (
	"testing"

	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func setup() {
	//md := &clients.MockDocker{}
}

func TestCorrectlyGeneratesProviders(t *testing.T) {
	n1 := &config.Network{Name: "network1"}
	c1 := &config.Container{Name: "container1", NetworkRef: n1}
	cl1 := &config.Cluster{Name: "cluster1", NetworkRef: n1}
	h1 := &config.Helm{Name: "helm1", ClusterRef: cl1}
	i1 := &config.Ingress{Name: "ingress1", TargetRef: cl1}

	c, _ := config.New()
	c.Containers = []*config.Container{c1}
	c.Clusters = []*config.Cluster{cl1}
	c.Networks = []*config.Network{n1}
	c.Ingresses = []*config.Ingress{i1}
	c.HelmCharts = []*config.Helm{h1}

	cl := &Clients{}

	// process the config
	oc := generateProviders(c, cl, hclog.NewNullLogger())

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
