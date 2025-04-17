package k8s

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	k8scli "github.com/instruqt/jumppad/pkg/clients/k8s"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/config/resources/healthcheck"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupK8sConfig(t *testing.T) (*k8scli.MockKubernetes, *ConfigProvider) {
	mk := &k8scli.MockKubernetes{}
	mk.On("SetConfig", mock.Anything).Return(nil)
	mk.On("Apply", mock.Anything, mock.Anything).Return(nil)
	mk.On("Delete", mock.Anything, mock.Anything).Return(nil)

	// create the test files
	d := t.TempDir()
	os.WriteFile(fmt.Sprintf("%s/testfile1", d), []byte("test1"), 0644)
	os.WriteFile(fmt.Sprintf("%s/testfile2", d), []byte("test2"), 0644)

	c := Cluster{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testcluster"}}}
	kc := Config{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "config"}}}
	kc.Cluster = c
	kc.Paths = []string{
		fmt.Sprintf("%s/testfile1", d),
		fmt.Sprintf("%s/testfile2", d),
	}

	p := &ConfigProvider{&kc, mk, logger.NewTestLogger(t)}

	return mk, p
}

func TestCreatesCorrectly(t *testing.T) {
	mk, p := setupK8sConfig(t)

	err := p.Create(context.Background())
	assert.NoError(t, err)

	//_, destPath, _ := utils.CreateKubeConfigPath("testcluster")
	//mk.AssertCalled(t, "SetConfig", destPath)
	mk.AssertCalled(t, "Apply", p.config.Paths, p.config.WaitUntilReady)
}

func TestRunsHealthChecks(t *testing.T) {
	mk, p := setupK8sConfig(t)
	p.config.HealthCheck = &healthcheck.HealthCheckKubernetes{
		Pods:    []string{"app=mine"},
		Timeout: "60s",
	}
	mk.On("HealthCheckPods", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := p.Create(context.Background())
	assert.NoError(t, err)

	mk.AssertCalled(t, "HealthCheckPods", mock.Anything, []string{"app=mine"}, 60*time.Second)
}

func TestHealthCheckFailReturnsError(t *testing.T) {
	mk, p := setupK8sConfig(t)
	p.config.HealthCheck = &healthcheck.HealthCheckKubernetes{
		Pods:    []string{"app=mine"},
		Timeout: "60s",
	}
	mk.On("HealthCheckPods", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create(context.Background())
	assert.Error(t, err)

	mk.AssertCalled(t, "HealthCheckPods", mock.Anything, []string{"app=mine"}, 60*time.Second)
}

func TestCreateSetupErrorReturnsError(t *testing.T) {
	mk, p := setupK8sConfig(t)
	testutils.RemoveOn(&mk.Mock, "SetConfig")
	mk.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestDestroysCorrectly(t *testing.T) {
	mk, p := setupK8sConfig(t)

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)

	mk.AssertCalled(t, "Delete", p.config.Paths)
}

func TestDestroySetupErrorReturnsError(t *testing.T) {
	mk, p := setupK8sConfig(t)
	testutils.RemoveOn(&mk.Mock, "SetConfig")
	mk.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background(), false)
	assert.Error(t, err)
}

func TestRefreshWithChangedFileReapplies(t *testing.T) {
	mk, p := setupK8sConfig(t)

	// create first to set checksums
	err := p.Create(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Apply", 1)

	// change the file
	os.WriteFile(p.config.Paths[0], []byte("test3"), 0644)

	err = p.Refresh(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Apply", 2)
}

func TestRefreshWithRemovedFileDeletes(t *testing.T) {
	mk, p := setupK8sConfig(t)

	// create first to set checksums
	err := p.Create(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Apply", 1)

	// change the file
	p.config.Paths = p.config.Paths[1:]

	err = p.Refresh(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Delete", 1)
}

func TestRefreshWithAddedFileApplies(t *testing.T) {
	mk, p := setupK8sConfig(t)

	// create first to set checksums
	err := p.Create(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Apply", 1)

	// change the file
	d := t.TempDir()
	os.WriteFile(fmt.Sprintf("%s/testfile3", d), []byte("test3"), 0644)
	p.config.Paths = p.config.Paths[1:]

	err = p.Refresh(context.Background())
	assert.NoError(t, err)

	mk.AssertNumberOfCalls(t, "Apply", 2)
}
