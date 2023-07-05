package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	clientmocks "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/mocks"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	gvm "github.com/shipyard-run/version-manager"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

type runMocks struct {
	engine    *mocks.Engine
	getter    *clientmocks.Getter
	http      *clientmocks.MockHTTP
	system    *clientmocks.System
	vm        *gvm.MockVersions
	connector *clients.ConnectorMock
}

func setupRun(t *testing.T, timeout string) (*cobra.Command, *runMocks) {
	mockHTTP := &clientmocks.MockHTTP{}
	mockHTTP.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockGetter := &clientmocks.Getter{}
	mockGetter.On("Get", mock.Anything, mock.Anything).Return(nil)
	mockGetter.On("SetForce", mock.Anything)

	mockSystem := &clientmocks.System{}
	mockSystem.On("OpenBrowser", mock.Anything).Return(nil)
	mockSystem.On("Preflight").Return(nil)
	mockSystem.On("PromptInput", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("")
	mockSystem.On("CheckVersion", mock.Anything).Return("", false)

	mockTasks := &clients.MockContainerTasks{}
	mockTasks.On("SetForcePull", mock.Anything)

	mockConnector := &clients.ConnectorMock{}
	mockConnector.On("GetLocalCertBundle", mock.Anything).Return(
		&clients.CertBundle{},
		nil,
	)

	mockConnector.On("GenerateLocalCertBundle", mock.Anything).Return(
		&clients.CertBundle{},
		nil,
	)

	mockConnector.On("IsRunning").Return(
		false,
	)

	mockConnector.On("Start", mock.Anything).Return(
		nil,
	)

	clients := &jumppad.Clients{
		HTTP:           mockHTTP,
		Getter:         mockGetter,
		Browser:        mockSystem,
		ContainerTasks: mockTasks,
		Connector:      mockConnector,
	}

	mockEngine := &mocks.Engine{}
	mockEngine.On("ParseConfigWithVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockEngine.On("ApplyWithVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockEngine.On("GetClients", mock.Anything).Return(clients)
	mockEngine.On("ResourceCountForType", mock.Anything).Return(0)

	bp := config.Blueprint{BrowserWindows: []string{"http://localhost", "http://localhost2"}}

	if timeout != "" {
		bp.HealthCheckTimeout = timeout
	}

	mockEngine.On("Blueprint").Return(&bp)

	vm := &gvm.MockVersions{}
	vm.On("ListInstalledVersions", mock.Anything).Return(nil, nil)
	vm.On("GetLatestReleaseURL", mock.Anything).Return("v1.0.0", "http://download.com", nil)

	rm := &runMocks{
		engine:    mockEngine,
		getter:    mockGetter,
		http:      mockHTTP,
		system:    mockSystem,
		vm:        vm,
		connector: mockConnector,
	}

	cmd := newRunCmd(mockEngine, mockGetter, mockHTTP, mockSystem, vm, mockConnector, hclog.Default())
	cmd.SetOut(bytes.NewBuffer([]byte("")))

	return cmd, rm
}

func TestRunSetsForceOnGetter(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.Flags().Set("force-update", "true")

	err := rf.Execute()
	assert.NoError(t, err)

	rm.getter.AssertCalled(t, "SetForce", true)
}

func TestRunPreflightsSystem(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.system.AssertCalled(t, "Preflight")
}

func TestRunOtherVersionChecksInstalledVersions(t *testing.T) {
	version := "v0.0.99"
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})
	rf.Flags().Set("version", version)

	err := rf.Execute()
	assert.NoError(t, err)

	rm.vm.AssertCalled(t, "ListInstalledVersions", version)
	rm.vm.AssertCalled(t, "GetLatestReleaseURL", version)
}

func TestRunOtherVersionPromptsInstallWhenNotInstalled(t *testing.T) {
	version := "v0.0.99"
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})
	rf.Flags().Set("version", version)

	err := rf.Execute()
	assert.NoError(t, err)

	rm.vm.AssertCalled(t, "ListInstalledVersions", version)
	rm.vm.AssertCalled(t, "GetLatestReleaseURL", version)
}

func TestRunChecksForCertBundle(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.connector.AssertCalled(t, "GetLocalCertBundle", mock.Anything)
}

func TestRunNotGeneratesCertBundleWhenExist(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.connector.AssertNotCalled(t, "GenerateLocalCertBundle", mock.Anything)
}

func TestRunGeneratesCertBundleWhenNotExist(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.connector.Mock, "GetLocalCertBundle")
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(nil, fmt.Errorf("boom")).Once()
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(clients.CertBundle{}, nil).Once()

	err := rf.Execute()
	assert.NoError(t, err)

	rm.connector.AssertCalled(t, "GenerateLocalCertBundle", mock.Anything)
}

func TestRunStartsConnectorWhenNotRunning(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.connector.AssertCalled(t, "Start", mock.Anything)
}

func TestRunDoesNotStartsConnectorWhenRunning(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.connector.Mock, "IsRunning")
	rm.connector.On("IsRunning", mock.Anything).Return(true).Once()

	err := rf.Execute()
	assert.NoError(t, err)

	rm.connector.AssertNotCalled(t, "Start", mock.Anything)
}

func TestRunConnectorStartErrorWhenGetCertBundleFails(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.connector.Mock, "GetLocalCertBundle")
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(clients.CertBundle{}, nil).Once()
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(nil, fmt.Errorf("boom")).Once()

	err := rf.Execute()
	assert.Error(t, err)

	rm.connector.AssertNotCalled(t, "Start", mock.Anything)
}

func TestRunSetsDestinationFromArgsWhenPresent(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", "/tmp", mock.Anything, mock.Anything)
}

func TestRunSetsVariablesFileReturnsErrorWhenMissing(t *testing.T) {
	rf, _ := setupRun(t, "")
	rf.SetArgs([]string{"--vars-file=./vars.file", "/tmp"})

	err := rf.Execute()
	assert.Error(t, err)
}

func TestRunSetsVariablesFileWhenPresent(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "*.vars")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"--vars-file=" + tmpFile.Name(), "/tmp"})

	err = rf.Execute()
	assert.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", "/tmp", mock.Anything, tmpFile.Name())
}

func TestRunSetsDestinationToDownloadedBlueprintFromArgsWhenRemote(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", filepath.Join(utils.JumppadHome(), "blueprints/github.com/shipyard-run/blueprints/vault-k8s"), mock.Anything, mock.Anything)
}

func TestRunFetchesBlueprint(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{bpf})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.getter.AssertCalled(t, "Get", bpf, mock.Anything)
}

func TestRunFetchesBlueprintErrorReturnsError(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{bpf})

	removeOn(&rm.getter.Mock, "Get")
	rm.getter.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.Error(t, err)
}

func TestRunOpensBrowserWindow(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 2)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", []int{200}, 30*time.Second)
}

func TestRunOpensBrowserWindowWithCustomTimeout(t *testing.T) {
	rf, rm := setupRun(t, "60s")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 2)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", []int{200}, 60*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", []int{200}, 60*time.Second)
}

func TestRunOpensBrowserWindowWithInvalidTimeout(t *testing.T) {
	rf, rm := setupRun(t, "6e")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 2)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", []int{200}, 30*time.Second)
}

func TestRunOpensBrowserWindowForResources(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.engine.Mock, "ApplyWithVariables")

	// should open
	d := config.NewDocs("test")
	d.OpenInBrowser = true

	// should open
	i := config.NewIngress("test")
	i.Source.Driver = config.IngressSourceLocal
	i.Source.Config.Port = "8080"
	i.Source.Config.OpenInBrowser = "/"

	// should open
	c := config.NewContainer("test")
	c.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: "https://test.container.jumppad.dev:8080"}}

	// should not be opened
	c2 := config.NewContainer("test2")
	c2.Ports = []config.Port{config.Port{OpenInBrowser: ""}}

	// should not be opened
	i2 := config.NewIngress("test")
	i.Source.Driver = config.IngressSourceLocal
	i2.Source.Config.Port = "8080"

	// should not be opened
	d2 := config.NewDocs("test2")

	// should be opened
	n1 := config.NewNomadCluster("test")
	n1.OpenInBrowser = true
	nomadConfig, _ := utils.GetClusterConfig("nomad_cluster.test")

	rm.engine.On("ApplyWithVariables", mock.Anything, mock.Anything, mock.Anything).Return(
		[]config.Resource{d, i, c, d2, i2, c2, n1},
		nil,
	)

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 6)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 6)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://test.ingress.jumppad.dev:8080/", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "https://test.container.jumppad.dev:8080", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://test.docs.jumppad.dev:80", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", fmt.Sprintf("http://server.test.nomad-cluster.jumppad.dev:%d/", nomadConfig.APIPort), []int{200}, 30*time.Second)
}

func TestRunDoesNotOpensBrowserWindowWhenCheckError(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.http.Mock, "HealthCheckHTTP")
	rm.http.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.NoError(t, err)

	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 0)
}
