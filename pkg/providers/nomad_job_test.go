package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNomadJobMocks() (*config.NomadJob, *mocks.MockNomad) {
	// copy the config
	cc := *clusterNomadConfig
	cn := *clusterNetwork
	jc := *nomadJob

	c := config.New()
	c.AddResource(&cc)
	c.AddResource(&cn)
	c.AddResource(&jc)

	mh := &mocks.MockNomad{}
	mh.On("SetConfig", mock.Anything).Return(nil)
	mh.On("Create", mock.Anything, mock.Anything).Return(nil)

	return &jc, mh
}

var nomadJob = &config.NomadJob{
	ResourceInfo: config.ResourceInfo{Name: "test", Type: config.TypeNomadJob},
	Cluster:      "nomad_cluster.test",
	Paths:        []string{"./example.nomad"},
}

func TestNomadJobWithNonExistentClusterReturnsError(t *testing.T) {
	jc, mh := setupNomadJobMocks()
	jc.Config.Resources = jc.Config.Resources[1:]

	p := NewNomadJob(jc, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestNomadJobUnableToLoadConfigReturnsError(t *testing.T) {
	jc, mh := setupNomadJobMocks()
	jc.Config.Resources = jc.Config.Resources[1:]

	removeOn(&mh.Mock, "SetConfig")
	mh.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	p := NewNomadJob(jc, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}
func TestNomadJobCreateReturnsError(t *testing.T) {
	jc, mh := setupNomadJobMocks()
	removeOn(&mh.Mock, "Create")
	mh.On("Create", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewNomadJob(jc, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestNomadJobValidatesConfig(t *testing.T) {
	jc, mh := setupNomadJobMocks()

	p := NewNomadJob(jc, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
}
