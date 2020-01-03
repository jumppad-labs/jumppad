package providers

import (
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
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
	c := &config.Cluster{Name: "testcluster"}
	kc := &config.K8sConfig{ClusterRef: c, Paths: paths}
	mk, p := setupK8sConfig(kc)

	err := p.Create()
	assert.NoError(t, err)

	_, destPath, _ := CreateKubeConfigPath(c.Name)
	mk.AssertCalled(t, "SetConfig", destPath)
	mk.AssertCalled(t, "Apply", paths, kc.WaitUntilReady)
}

func TestDestroysCorrectly(t *testing.T) {
	paths := []string{"/tmp/something"}
	c := &config.Cluster{Name: "testcluster"}
	kc := &config.K8sConfig{ClusterRef: c, Paths: paths}
	mk, p := setupK8sConfig(kc)

	err := p.Destroy()
	assert.NoError(t, err)

	mk.AssertCalled(t, "Delete", paths)
}
