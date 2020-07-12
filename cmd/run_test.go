package cmd

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/nicholasjackson/version-manager"
	clientmocks "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/shipyard/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRun(t *testing.T, timeout string) (*cobra.Command, *mocks.Engine, *clientmocks.Getter, *clientmocks.MockHTTP, *clientmocks.System) {

	mockHTTP := &clientmocks.MockHTTP{}
	mockHTTP.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(nil)

	mockGetter := &clientmocks.Getter{}
	mockGetter.On("Get", mock.Anything, mock.Anything).Return(nil)
	mockGetter.On("SetForce", mock.Anything)

	mockBrowser := &clientmocks.System{}
	mockBrowser.On("OpenBrowser", mock.Anything).Return(nil)
	mockBrowser.On("Preflight").Return(nil)
	mockBrowser.On("CheckVersion", mock.Anything).Return("", false)

	mockTasks := &clientmocks.MockContainerTasks{}
	mockTasks.On("SetForcePull", mock.Anything)

	clients := &shipyard.Clients{
		HTTP:           mockHTTP,
		Getter:         mockGetter,
		Browser:        mockBrowser,
		ContainerTasks: mockTasks,
	}

	mockEngine := &mocks.Engine{}
	mockEngine.On("Apply", mock.Anything).Return(nil, nil)
	mockEngine.On("GetClients", mock.Anything).Return(clients)

	bp := config.Blueprint{BrowserWindows: []string{"http://localhost", "http://localhost2"}}

	if timeout != "" {
		bp.HealthCheckTimeout = timeout
	}

	mockEngine.On("Blueprint").Return(&bp)

	vm := gvm.New(nil)

	return newRunCmd(mockEngine, mockGetter, mockHTTP, mockBrowser, vm, hclog.Default()), mockEngine, mockGetter, mockHTTP, mockBrowser
}

func TestRunSetsForceOnGetter(t *testing.T) {
	rf, _, mg, _, _ := setupRun(t, "")
	rf.Flags().Set("force-update", "true")

	err := rf.Execute()
	assert.NoError(t, err)

	mg.AssertCalled(t, "SetForce", true)
}

func TestRunPreflightsSystem(t *testing.T) {
	rf, _, _, _, mb := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	mb.AssertCalled(t, "Preflight")
}

func TestRunSetsDestinationFromArgsWhenPresent(t *testing.T) {
	rf, me, _, _, _ := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	me.AssertCalled(t, "Apply", "/tmp")
}

func TestRunSetsDestinationToDownloadedBlueprintFromArgsWhenRemote(t *testing.T) {
	rf, me, _, _, _ := setupRun(t, "")
	rf.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})

	err := rf.Execute()
	assert.NoError(t, err)

	me.AssertCalled(t, "Apply", filepath.Join(utils.ShipyardHome(), "blueprints/github.com/shipyard-run/blueprints/vault-k8s"))
}

func TestRunFetchesBlueprint(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, _, mg, _, _ := setupRun(t, "")
	rf.SetArgs([]string{bpf})

	err := rf.Execute()
	assert.NoError(t, err)

	mg.AssertCalled(t, "Get", bpf, mock.Anything)
}

func TestRunFetchesBlueprintErrorReturnsError(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, _, mb, _, _ := setupRun(t, "")
	rf.SetArgs([]string{bpf})

	removeOn(&mb.Mock, "Get")
	mb.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.Error(t, err)
}

func TestRunOpensBrowserWindow(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	mb.AssertNumberOfCalls(t, "OpenBrowser", 2)

	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 30*time.Second)
	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 30*time.Second)
}

func TestRunOpensBrowserWindowWithCustomTimeout(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t, "60s")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	mb.AssertNumberOfCalls(t, "OpenBrowser", 2)

	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 60*time.Second)
	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 60*time.Second)
}

func TestRunOpensBrowserWindowWithInvalidTimeout(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t, "6e")
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	mb.AssertNumberOfCalls(t, "OpenBrowser", 2)

	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost", 30*time.Second)
	mh.AssertCalled(t, "HealthCheckHTTP", "http://localhost2", 30*time.Second)
}

func TestRunOpensBrowserWindowForResources(t *testing.T) {
	rf, me, _, mh, mb := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&me.Mock, "Apply")

	d := config.NewDocs("test")
	d.OpenInBrowser = true

	i := config.NewIngress("test")
	i.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: "/"}}

	c := config.NewContainer("test")
	c.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: "/"}}

	// should not be opened
	d2 := config.NewDocs("test")

	i2 := config.NewIngress("test")
	i2.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: ""}}

	c2 := config.NewContainer("test2")
	c2.Ports = []config.Port{config.Port{OpenInBrowser: ""}}

	me.On("Apply", mock.Anything).Return(
		[]config.Resource{d, i, c, d2, i2, c2},
		nil,
	)

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 5)
	mb.AssertNumberOfCalls(t, "OpenBrowser", 5)
}

func TestRunDoesNotOpensBrowserWindowWhenCheckError(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t, "")
	rf.SetArgs([]string{"/tmp"})

	removeOn(&mh.Mock, "HealthCheckHTTP")
	mh.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.NoError(t, err)

	mb.AssertNumberOfCalls(t, "OpenBrowser", 0)
}
