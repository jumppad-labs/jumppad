package providers

import (
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupK8sConfig(c *config.K8sConfig) (*clients.MockKubernetes, *K8sConfig) {
	mk := &clients.MockKubernetes{}
	mk.On("SetConfig", mock.Anything).Return(nil)
	mk.On("Apply", mock.Anything, mock.Anything).Return(nil)
	mk.On("Delete", mock.Anything, mock.Anything).Return(nil)

	p := NewK8sConfig(c, mk, hclog.Default())

	return mk, p
}

func TestCreatesConfigCorrectly(t *testing.T) {
	paths := []string{"/tmp/something"}
	c := config.NewK8sCluster("testcluster")
	kc := config.NewK8sConfig("config")
	kc.Cluster = "k8s_cluster.testcluster"
	kc.Paths = paths
	mk, p := setupK8sConfig(kc)

	err := p.Create()
	assert.NoError(t, err)

	_, destPath, _ := utils.CreateKubeConfigPath(c.Name)
	mk.AssertCalled(t, "SetConfig", destPath)
	mk.AssertCalled(t, "Apply", paths, kc.WaitUntilReady)
}

func TestDestroysCorrectly(t *testing.T) {
	//skip this test as functionality has been removed until we implement a DAG
	t.SkipNow()

	paths := []string{"/tmp/something"}
	kc := config.NewK8sConfig("config")
	kc.Cluster = "k8s_cluster.testcluster"
	kc.Paths = paths
	mk, p := setupK8sConfig(kc)

	err := p.Destroy()
	assert.NoError(t, err)

	mk.AssertCalled(t, "Delete", paths)
}
