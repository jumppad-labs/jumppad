package providers

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNomadMocks() (config.Cluster, *mocks.MockContainerTasks, *mocks.MockHTTP) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return("", nil)
	md.On("CreateVolume", mock.Anything).Return("", nil)
	md.On("CreateContainer", mock.Anything).Return("", nil)

	hc := &mocks.MockHTTP{}

	nc := testNomadClusterConfig

	return nc, md, hc
}

func TestNomadChecksClusterExists(t *testing.T) {
	nc, md, hc := setupNomadMocks()
	p := NewCluster(nc, md, nil, hc, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", nc.Name, nc.NetworkRef.Name)
}

var testNomadClusterConfig = config.Cluster{
	Name:       "nomad_test",
	Driver:     "nomad",
	NetworkRef: &config.Network{Name: "cloud"},
}
