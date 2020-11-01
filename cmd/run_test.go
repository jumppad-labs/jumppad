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
	clientmocks "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/shipyard/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"
	gvm "github.com/shipyard-run/version-manager"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type runMocks struct {
	engine *mocks.Engine
	getter *clientmocks.Getter
	http   *clientmocks.MockHTTP
	system *clientmocks.System
	vm     *gvm.MockVersions
}

func setupRun(t *testing.T, timeout string) (*cobra.Command, *runMocks) {
	mockHTTP := &clientmocks.MockHTTP{}
	mockHTTP.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(nil)

	mockGetter := &clientmocks.Getter{}
	mockGetter.On("Get", mock.Anything, mock.Anything).Return(nil)
	mockGetter.On("SetForce", mock.Anything)

	mockSystem := &clientmocks.System{}
	mockSystem.On("OpenBrowser", mock.Anything).Return(nil)
	mockSystem.On("Preflight").Return(nil)
	mockSystem.On("PromptInput", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("")
	mockSystem.On("CheckVersion", mock.Anything).Return("", false)

	mockTasks := &clientmocks.MockContainerTasks{}
	mockTasks.On("SetForcePull", mock.Anything)

	clients := &shipyard.Clients{
		HTTP:           mockHTTP,
		Getter:         mockGetter,
		Browser:        mockSystem,
		ContainerTasks: mockTasks,
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
		engine: mockEngine,
		getter: mockGetter,
		http:   mockHTTP,
		system: mockSystem,
		vm:     vm,
	}

	cmd := newRunCmd(mockEngine, mockGetter, mockHTTP, mockSystem, vm, hclog.Default())
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

	rm.engine.AssertCalled(t, "ApplyWithVariables", filepath.Join(utils.ShipyardHome(), "blueprints/github.com/shipyard-run/blueprints/vault-k8s"), mock.Anything, mock.Anything)
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

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 30*time.Second)
}

func TestRunOpensBrowserWindowWithCustomTimeout(t *testing.T) {
	rf, rm := setupRun(t, "60s")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 2)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 60*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 60*time.Second)
}

func TestRunOpensBrowserWindowWithInvalidTimeout(t *testing.T) {
	rf, rm := setupRun(t, "6e")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 2)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 30*time.Second)
}

func TestRunOpensBrowserWindowForResources(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.engine.Mock, "ApplyWithVariables")

	d := config.NewDocs("test")
	d.OpenInBrowser = true

	i := config.NewIngress("test")
	i.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: "/"}}

	c := config.NewContainer("test")
	c.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: "https://test.container.shipyard.run:8080"}}

	// should not be opened
	d2 := config.NewDocs("test")

	i2 := config.NewIngress("test")
	i2.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: ""}}

	c2 := config.NewContainer("test2")
	c2.Ports = []config.Port{config.Port{OpenInBrowser: ""}}

	rm.engine.On("ApplyWithVariables", mock.Anything, mock.Anything, mock.Anything).Return(
		[]config.Resource{d, i, c, d2, i2, c2},
		nil,
	)

	err := rf.Execute()
	assert.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 5)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 5)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://test.ingress.shipyard.run:8080/", 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "https://test.container.shipyard.run:8080", 30*time.Second)
}

func TestRunDoesNotOpensBrowserWindowWhenCheckError(t *testing.T) {
	rf, rm := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&rm.http.Mock, "HealthCheckHTTP")
	rm.http.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.NoError(t, err)

	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 0)
}
