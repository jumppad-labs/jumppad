package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupHelm() (*clients.MockHelm, *clients.MockKubernetes, *config.Config, *Helm) {
	mh := &clients.MockHelm{}
	mh.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	kc := &clients.MockKubernetes{}
	kc.On("SetConfig", mock.Anything).Return(nil)

	cl := config.NewK8sCluster("tester")
	ch := config.NewHelm("test")
	ch.Cluster = "k8s_cluster.tester"

	c := config.New()
	c.AddResource(cl)
	c.AddResource(ch)

	p := NewHelm(ch, kc, mh, hclog.NewNullLogger())

	return mh, kc, c, p
}

func TestHelmCreateCantFindClusterReturnsError(t *testing.T) {
	_, _, c, p := setupHelm()
	c.RemoveResource(c.Resources[0])

	err := p.Create()
	assert.Error(t, err)
}

func TestHelmCreateSetsConfig(t *testing.T) {
	_, kc, _, p := setupHelm()

	err := p.Create()
	assert.NoError(t, err)

	_, fp, _ := utils.CreateKubeConfigPath("tester")
	kc.AssertCalled(t, "SetConfig", fp)
}

func TestHelmCreateConfigSetFailReturnsError(t *testing.T) {
	_, kc, _, p := setupHelm()
	removeOn(&kc.Mock, "SetConfig")
	kc.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}

func TestHelmCreateCallsCreate(t *testing.T) {
	hm, _, _, p := setupHelm()

	err := p.Create()
	assert.NoError(t, err)

	hm.AssertCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHelmCreateCallCreateFailReturnsError(t *testing.T) {
	hm, _, _, p := setupHelm()
	removeOn(&hm.Mock, "Create")
	hm.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}
