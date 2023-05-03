package providers

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupHelm() (*mocks.MockHelm, *clients.MockKubernetes, *mocks.Getter, *config.Config, *Helm) {
	mh := &mocks.MockHelm{}
	mh.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mh.On("Destroy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mh.On("UpsertChartRepository", mock.Anything, mock.Anything).Return(nil)

	kc := &clients.MockKubernetes{}
	kc.On("SetConfig", mock.Anything).Return(nil)
	kc.On("HealthCheckPods", mock.Anything, mock.Anything).Return(nil)

	mg := &mocks.Getter{}
	mg.On("Get", mock.Anything, mock.Anything).Return(nil)

	cl := config.NewK8sCluster("tester")
	ch := config.NewHelm("test")
	ch.ChartName = "test"
	ch.Cluster = "k8s_cluster.tester"
	ch.SkipCRDs = true

	c := config.New()
	c.AddResource(cl)
	c.AddResource(ch)

	p := NewHelm(ch, kc, mh, mg, hclog.NewNullLogger())

	return mh, kc, mg, c, p
}

func TestHelmCreateCantFindClusterReturnsError(t *testing.T) {
	_, _, _, c, p := setupHelm()
	c.RemoveResource(c.Resources[0])

	err := p.Create()
	assert.Error(t, err)
}

func TestHelmCreateSantisesChartName(t *testing.T) {
	mh, _, _, c, p := setupHelm()
	hc, _ := c.FindResource("helm.test")
	hc.(*config.Helm).ChartName = "chart_test"

	err := p.Create()
	assert.NoError(t, err)

	mh.AssertCalled(t, "Create", mock.Anything, "chart-test", mock.Anything, mock.Anything, true, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHelmCreateGetsHelmRepo(t *testing.T) {
	mh, _, mg, c, p := setupHelm()
	hc, _ := c.FindResource("helm.test")

	hc.(*config.Helm).Repository = &config.HelmRepository{URL: "http://something.com", Name: "hashicorp"}
	hc.(*config.Helm).Chart = "hashicorp/vault"
	hc.(*config.Helm).Version = "v1.0.0"

	err := p.Create()
	assert.NoError(t, err)

	mg.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
	mh.AssertCalled(t, "UpsertChartRepository", "hashicorp", "http://something.com")
	mh.AssertCalled(t, "Create", mock.Anything, "test", mock.Anything, mock.Anything, true, "hashicorp/vault", "v1.0.0", mock.Anything, mock.Anything)
}

func TestHelmCreateGetsRemoteRepo(t *testing.T) {
	mh, _, mg, c, p := setupHelm()
	hc, _ := c.FindResource("helm.test")
	hc.(*config.Helm).Chart = "github.com/shipyard-run/blueprints//vault-k8s"

	helmFolder := filepath.Join(utils.ShipyardHome(), "helm_charts", strings.Replace(hc.(*config.Helm).Chart, "//", "/", -1))

	err := p.Create()
	assert.NoError(t, err)

	mg.AssertCalled(t, "Get", mock.Anything, helmFolder)
	mh.AssertCalled(t, "Create", mock.Anything, "test", mock.Anything, mock.Anything, true, helmFolder, "", mock.Anything, mock.Anything)
}

func TestHelmCreateSetsConfig(t *testing.T) {
	_, kc, mg, _, p := setupHelm()

	err := p.Create()
	assert.NoError(t, err)

	_, fp, _ := utils.CreateKubeConfigPath("tester")
	kc.AssertCalled(t, "SetConfig", fp)
	mg.AssertNotCalled(t, "Get")
}

func TestHelmCreateConfigSetFailReturnsError(t *testing.T) {
	_, kc, _, _, p := setupHelm()
	removeOn(&kc.Mock, "SetConfig")
	kc.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}

func TestHelmCreateCallsCreateWithDefaultNamespace(t *testing.T) {
	hm, _, _, _, p := setupHelm()

	err := p.Create()
	assert.NoError(t, err)

	hm.AssertCalled(
		t,
		"Create",
		mock.Anything,
		p.config.Name,
		"default",
		false,
		true,
		p.config.Chart,
		"",
		p.config.Values,
		p.config.ValuesString,
	)
}

func TestHelmCreateCallsCreateWithCustomNamespace(t *testing.T) {
	hm, _, _, _, p := setupHelm()
	p.config.Namespace = "custom"

	err := p.Create()
	assert.NoError(t, err)

	hm.AssertCalled(
		t,
		"Create",
		mock.Anything,
		p.config.Name,
		"custom",
		p.config.CreateNamespace,
		true,
		p.config.Chart,
		"",
		p.config.Values,
		p.config.ValuesString,
	)
}

func TestHelmCreateCallCreateFailRetries(t *testing.T) {
	hm, _, _, _, p := setupHelm()
	p.config.Retry = 2

	removeOn(&hm.Mock, "Create")
	hm.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, true, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(fmt.Errorf("boom"))
	hm.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, true, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

	err := p.Create()
	assert.NoError(t, err)
	hm.AssertNumberOfCalls(t, "Create", 2)
}

func TestHelmCreateCallCreateFailReturnsError(t *testing.T) {
	hm, _, _, _, p := setupHelm()

	removeOn(&hm.Mock, "Create")
	hm.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, true, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}

func TestHelmDoesNotHealthChecksPodswhenNotSet(t *testing.T) {
	_, kc, _, _, p := setupHelm()

	err := p.Create()
	assert.NoError(t, err)

	kc.AssertNotCalled(t, "HealthCheckPods", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHelmHealthChecksPodswhenSet(t *testing.T) {
	_, kc, _, _, p := setupHelm()
	p.config.HealthCheck = &config.HealthCheck{Timeout: "1s", Pods: []string{"consul=release"}}

	err := p.Create()
	assert.NoError(t, err)

	kc.AssertCalled(t, "HealthCheckPods", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHelmCreateHealthCheckPodsFailReturnsError(t *testing.T) {
	_, kc, _, _, p := setupHelm()
	p.config.HealthCheck = &config.HealthCheck{Timeout: "1s", Pods: []string{"consul=release"}}
	removeOn(&kc.Mock, "HealthCheckPods")
	kc.On("HealthCheckPods", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Create()
	assert.Error(t, err)
}
func TestHelmDestroyCantFindClusterReturnsError(t *testing.T) {
	_, _, _, c, p := setupHelm()
	c.RemoveResource(c.Resources[0])

	err := p.Destroy()
	assert.Error(t, err)
}

func TestHelmDestroyCallsDestroyWithDefaultNamespace(t *testing.T) {
	hm, _, _, _, p := setupHelm()

	err := p.Destroy()
	assert.NoError(t, err)
	hm.AssertCalled(t, "Destroy", mock.Anything, mock.Anything, "default")
}

func TestHelmDestroyWithErrorSwallowsError(t *testing.T) {
	hm, _, _, _, p := setupHelm()
	p.config.Namespace = "custom"
	removeOn(&hm.Mock, "Destroy")
	hm.On("Destroy", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy()
	assert.NoError(t, err)
	hm.AssertCalled(t, "Destroy", mock.Anything, mock.Anything, "custom")
}

func TestHelmDestroySantisesChartName(t *testing.T) {
	mh, _, _, c, p := setupHelm()
	hc, _ := c.FindResource("helm.test")
	hc.(*config.Helm).ChartName = "chart_test"

	err := p.Destroy()
	assert.NoError(t, err)

	mh.AssertCalled(t, "Destroy", mock.Anything, "chart-test", mock.Anything)
}
