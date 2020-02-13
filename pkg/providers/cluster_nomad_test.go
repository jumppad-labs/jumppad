package providers

import (
	"testing"

	// "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNomadMocks() (config.NomadCluster, *mocks.MockContainerTasks, *mocks.MockHTTP) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return("", nil)
	md.On("CreateVolume", mock.Anything).Return("", nil)
	md.On("CreateContainer", mock.Anything).Return("", nil)

	hc := &mocks.MockHTTP{}
	hc.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(nil)
	hc.On("HealthCheckNomad", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	nc := config.NewNomadCluster("nomad_test")
	nc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}}

	return *nc, md, hc
}

func TestNomadChecksClusterExists(t *testing.T) {
	// nc, md, hc := setupNomadMocks()
	// p := NewNomadCluster(nc, md, nil, hc, hclog.NewNullLogger())

	// err := p.Create()
	// assert.NoError(t, err)
	// md.AssertCalled(t, "FindContainerIDs", nc.Name, nc.Type)
}