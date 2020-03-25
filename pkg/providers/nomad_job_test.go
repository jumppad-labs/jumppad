package providers

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
)

func setupNomadJobMocks() (*config.NomadJob, clients.HTTP) {
	// copy the config
	cc := *clusterNomadConfig
	cn := *clusterNetwork
	jc := *nomadJob

	c := config.New()
	c.AddResource(&cc)
	c.AddResource(&cn)
	c.AddResource(&jc)

	mh := &mocks.MockHTTP{}

	return &jc, mh
}

var nomadJob = &config.NomadJob{
	ResourceInfo: config.ResourceInfo{Name: "test", Type: config.TypeNomadCluster},
	Cluster:      "nomad_cluster.test",
	Paths:        []string{"./example.nomad"},
}

func TestNomadJobValidatesConfig(t *testing.T) {
	jc, mh := setupNomadJobMocks()

	p := NewNomadJob(jc, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
}
