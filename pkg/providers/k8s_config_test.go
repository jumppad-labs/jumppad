package providers

import (
	"fmt"
	"testing"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupK8sConfig() (*clients.MockKubernetes, *K8sConfig) {
	mk := &clients.MockKubernetes{}
	mk.On("SetConfig", mock.Anything).Return(nil)
	mk.On("Apply", mock.Anything, mock.Anything).Return(nil)
	mk.On("Delete", mock.Anything, mock.Anything).Return(nil)

	c := config.NewK8sCluster("testcluster")
	kc := config.NewK8sConfig("config")
	kc.Cluster = "k8s_cluster.testcluster"
	kc.Paths = []string{"/tmp/something"}

	cc := config.New()
	cc.AddResource(kc)
	cc.AddResource(c)

	p := NewK8sConfig(kc, mk, hclog.Default())

	return mk, p
}

func TestCreatesCorrectly(t *testing.T) {
	mk, p := setupK8sConfig()

	err := p.Create()
	assert.NoError(t, err)

	_, destPath, _ := utils.CreateKubeConfigPath("testcluster")
	mk.AssertCalled(t, "SetConfig", destPath)
	mk.AssertCalled(t, "Apply", p.config.Paths, p.config.WaitUntilReady)
}

func TestRunsHealthChecks(t *testing.T) {
	mk, p := setupK8sConfig()
	p.config.HealthCheck = &config.HealthCheck{
		Pods:    []string{"app=mine"},
		Timeout: "60s",
	}
	mk.On("HealthCheckPods", mock.Anything, mock.Anything).Return(nil)

	err := p.Create()
	assert.NoError(t, err)

	mk.AssertCalled(t, "HealthCheckPods", []string{"app=mine"}, 60*time.Second)
}

func TestHealthCheckFailReturnsError(t *testing.T) {
	mk, p := setupK8sConfig()
	p.config.HealthCheck = &config.HealthCheck{
		Pods:    []string{"app=mine"},
		Timeout: "60s",
	}
	mk.On("HealthCheckPods", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)

	mk.AssertCalled(t, "HealthCheckPods", []string{"app=mine"}, 60*time.Second)
}

func TestCreateSetupErrorReturnsError(t *testing.T) {
	mk, p := setupK8sConfig()
	removeOn(&mk.Mock, "SetConfig")
	mk.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}

func TestCreateNoClusterErrorReturnsError(t *testing.T) {
	_, p := setupK8sConfig()
	p.config.Config.RemoveResource(p.config.Config.Resources[1])

	err := p.Create()
	assert.Error(t, err)
}

func TestDestroysCorrectly(t *testing.T) {
	mk, p := setupK8sConfig()

	err := p.Destroy()
	assert.NoError(t, err)

	mk.AssertCalled(t, "Delete", p.config.Paths)
}

func TestDestroySetupErrorReturnsError(t *testing.T) {
	mk, p := setupK8sConfig()
	removeOn(&mk.Mock, "SetConfig")
	mk.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy()
	assert.Error(t, err)
}
