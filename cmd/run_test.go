package cmd

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	clientmocks "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRun(t *testing.T) (*cobra.Command, *mocks.Engine, *clientmocks.Blueprints, *clientmocks.MockHTTP, *clientmocks.Browser) {
	mockEngine := &mocks.Engine{}
	mockEngine.On("Apply", mock.Anything).Return(nil, nil)
	mockEngine.On("Blueprint").Return(&config.Blueprint{BrowserWindows: []string{"http://localhost", "http://localhost2"}})

	mockHTTP := &clientmocks.MockHTTP{}
	mockHTTP.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(nil)

	mockBlueprints := &clientmocks.Blueprints{}
	mockBlueprints.On("Get", mock.Anything, mock.Anything).Return(nil)

	mockBrowser := &clientmocks.Browser{}
	mockBrowser.On("Open", mock.Anything).Return(nil)

	return newRunCmd(mockEngine, mockBlueprints, mockHTTP, mockBrowser, hclog.Default()), mockEngine, mockBlueprints, mockHTTP, mockBrowser
}

func TestRunSetsDestinationFromArgsWhenPresent(t *testing.T) {
	rf, me, _, _, _ := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	me.AssertCalled(t, "Apply", "/tmp")
}

func TestRunSetsDestinationToDownloadedBlueprintFromArgsWhenRemote(t *testing.T) {
	rf, me, _, _, _ := setupRun(t)
	rf.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})

	err := rf.Execute()
	assert.NoError(t, err)

	me.AssertCalled(t, "Apply", filepath.Join(utils.ShipyardHome(), "blueprints/github.com/shipyard-run/blueprints/vault-k8s"))
}

func TestRunFetchesBlueprint(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, _, mg, _, _ := setupRun(t)
	rf.SetArgs([]string{bpf})

	err := rf.Execute()
	assert.NoError(t, err)

	mg.AssertCalled(t, "Get", bpf, mock.Anything)
}

func TestRunFetchesBlueprintErrorReturnsError(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, _, mb, _, _ := setupRun(t)
	rf.SetArgs([]string{bpf})

	removeOn(&mb.Mock, "Get")
	mb.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.Error(t, err)
}

func TestRunOpensBrowserWindow(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 2)
	mb.AssertNumberOfCalls(t, "Open", 2)
}

func TestRunOpensBrowserWindowForResources(t *testing.T) {
	rf, me, _, mh, mb := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	removeOn(&me.Mock, "Apply")

	d := config.NewDocs("test")
	d.OpenInBrowser = true

	i := config.NewIngress("test")
	i.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: true}}

	c := config.NewContainer("test")
	c.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: true}}

	d2 := config.NewDocs("test")

	i2 := config.NewIngress("test")
	i2.Ports = []config.Port{config.Port{Host: "8080", OpenInBrowser: false}}

	c2 := config.NewContainer("test")
	c2.Ports = []config.Port{config.Port{OpenInBrowser: true}}

	me.On("Apply", mock.Anything).Return(
		[]config.Resource{d, i, c, d2, i2, c2},
		nil,
	)

	err := rf.Execute()
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "HealthCheckHTTP", 5)
	mb.AssertNumberOfCalls(t, "Open", 5)
}

func TestRunDoesNotOpensBrowserWindowWhenCheckError(t *testing.T) {
	rf, _, _, mh, mb := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	removeOn(&mh.Mock, "HealthCheckHTTP")
	mh.On("HealthCheckHTTP", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	assert.NoError(t, err)

	mb.AssertNumberOfCalls(t, "Open", 0)
}
