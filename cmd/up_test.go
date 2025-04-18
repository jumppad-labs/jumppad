package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/instruqt/jumppad/pkg/clients"
	conmock "github.com/instruqt/jumppad/pkg/clients/connector/mocks"
	"github.com/instruqt/jumppad/pkg/clients/connector/types"
	cmock "github.com/instruqt/jumppad/pkg/clients/container/mocks"
	gettermock "github.com/instruqt/jumppad/pkg/clients/getter/mocks"
	httpmock "github.com/instruqt/jumppad/pkg/clients/http/mocks"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	systemmock "github.com/instruqt/jumppad/pkg/clients/system/mocks"
	"github.com/instruqt/jumppad/pkg/config/resources/blueprint"
	"github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/instruqt/jumppad/pkg/config/resources/docs"
	"github.com/instruqt/jumppad/pkg/config/resources/ingress"
	"github.com/instruqt/jumppad/pkg/config/resources/nomad"
	enginemocks "github.com/instruqt/jumppad/pkg/jumppad/mocks"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig"
	hcltypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type runMocks struct {
	engine    *enginemocks.Engine
	getter    *gettermock.Getter
	http      *httpmock.HTTP
	system    *systemmock.System
	tasks     *cmock.ContainerTasks
	connector *conmock.Connector
}

func setupRun(t *testing.T) (*cobra.Command, *runMocks) {
	mockContainer := &cmock.ContainerTasks{}
	mockContainer.On("SetForce", mock.Anything)

	mockHTTP := &httpmock.HTTP{}
	mockHTTP.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockGetter := &gettermock.Getter{}
	mockGetter.On("Get", mock.Anything, mock.Anything).Return(nil)
	mockGetter.On("SetForce", mock.Anything)

	mockSystem := &systemmock.System{}
	mockSystem.On("OpenBrowser", mock.Anything).Return(nil)
	mockSystem.On("Preflight").Return(nil)
	mockSystem.On("PromptInput", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("")
	mockSystem.On("CheckVersion", mock.Anything).Return("", false)

	mockConnector := &conmock.Connector{}
	mockConnector.On("GetLocalCertBundle", mock.Anything).Return(
		&types.CertBundle{},
		nil,
	)

	mockConnector.On("GenerateLocalCertBundle", mock.Anything).Return(
		&types.CertBundle{},
		nil,
	)

	mockConnector.On("IsRunning").Return(
		false,
	)

	mockConnector.On("Start", mock.Anything).Return(
		nil,
	)

	clients := &clients.Clients{
		HTTP:      mockHTTP,
		Getter:    mockGetter,
		Connector: mockConnector,
	}

	hclconfig := hclconfig.Config{}

	mockEngine := &enginemocks.Engine{}
	mockEngine.On("ParseConfigWithVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockEngine.On("ApplyWithVariables", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&hclconfig, nil)
	mockEngine.On("GetClients", mock.Anything).Return(clients)
	mockEngine.On("ResourceCountForType", mock.Anything).Return(0)

	bp := blueprint.Blueprint{}

	mockEngine.On("Blueprint").Return(&bp)

	rm := &runMocks{
		engine:    mockEngine,
		getter:    mockGetter,
		http:      mockHTTP,
		system:    mockSystem,
		connector: mockConnector,
		tasks:     mockContainer,
	}

	cmd := newRunCmd(mockEngine, mockContainer, mockGetter, mockHTTP, mockSystem, mockConnector, logger.NewTestLogger(t))
	cmd.SetOut(bytes.NewBuffer([]byte("")))

	return cmd, rm
}

func TestRunSetsForceOnClients(t *testing.T) {
	rf, rm := setupRun(t)
	rf.Flags().Set("no-browser", "true")
	rf.Flags().Set("force-update", "true")

	err := rf.Execute()
	require.NoError(t, err)

	rm.getter.AssertCalled(t, "SetForce", true)
	rm.tasks.AssertCalled(t, "SetForce", true)
}

func TestRunChecksForCertBundle(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	require.NoError(t, err)

	rm.connector.AssertCalled(t, "GetLocalCertBundle", mock.Anything)
}

func TestRunNotGeneratesCertBundleWhenExist(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	require.NoError(t, err)

	rm.connector.AssertNotCalled(t, "GenerateLocalCertBundle", mock.Anything)
}

func TestRunGeneratesCertBundleWhenNotExist(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	testutils.RemoveOn(&rm.connector.Mock, "GetLocalCertBundle")
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(nil, fmt.Errorf("boom")).Once()
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(&types.CertBundle{}, nil).Once()

	err := rf.Execute()
	require.NoError(t, err)

	rm.connector.AssertCalled(t, "GenerateLocalCertBundle", mock.Anything)
}

func TestRunStartsConnectorWhenNotRunning(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	require.NoError(t, err)

	rm.connector.AssertCalled(t, "Start", mock.Anything)
}

func TestRunDoesNotStartsConnectorWhenRunning(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	testutils.RemoveOn(&rm.connector.Mock, "IsRunning")
	rm.connector.On("IsRunning", mock.Anything).Return(true).Once()

	err := rf.Execute()
	require.NoError(t, err)

	rm.connector.AssertNotCalled(t, "Start", mock.Anything)
}

func TestRunConnectorStartErrorWhenGetCertBundleFails(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	testutils.RemoveOn(&rm.connector.Mock, "GetLocalCertBundle")
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(&types.CertBundle{}, nil).Once()
	rm.connector.On("GetLocalCertBundle", mock.Anything).Return(nil, fmt.Errorf("boom")).Once()

	err := rf.Execute()
	require.Error(t, err)

	rm.connector.AssertNotCalled(t, "Start", mock.Anything)
}

func TestRunSetsDestinationFromArgsWhenPresent(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	err := rf.Execute()
	require.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", mock.Anything, "/tmp", mock.Anything, mock.Anything)
}

func TestRunSetsVariablesFromFlag(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{
		"--var=abc=1234",
		"--var='foo=bar'",
		"--var=\"erik=smells\"",
		"--var=nic=cool=beans",
		"/tmp",
	})

	err := rf.Execute()
	require.NoError(t, err)

	args := rm.engine.Calls[0].Arguments[2]

	require.Equal(t, map[string]string{
		"abc":  "1234",
		"foo":  "bar",
		"erik": "smells",
		"nic":  "cool=beans",
	}, args)
}

func TestRunSetsVariablesFileReturnsErrorWhenMissing(t *testing.T) {
	rf, _ := setupRun(t)
	rf.SetArgs([]string{"--vars-file=./vars.file", "/tmp"})

	err := rf.Execute()
	require.Error(t, err)
}

func TestRunSetsVariablesFileWhenPresent(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "*.vars")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	rf, rm := setupRun(t)
	rf.SetArgs([]string{"--vars-file=" + tmpFile.Name(), "/tmp"})

	err = rf.Execute()
	require.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", mock.Anything, "/tmp", mock.Anything, tmpFile.Name())
}

func TestRunSetsDestinationToDownloadedBlueprintFromArgsWhenRemote(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"github.com/shipyard-run/blueprints//vault-k8s"})

	err := rf.Execute()
	require.NoError(t, err)

	rm.engine.AssertCalled(t, "ApplyWithVariables", mock.Anything, filepath.Join(utils.JumppadHome(), "blueprints/github.com/shipyard-run/blueprints/vault-k8s"), mock.Anything, mock.Anything)
}

func TestRunFetchesBlueprint(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, rm := setupRun(t)
	rf.SetArgs([]string{bpf})

	err := rf.Execute()
	require.NoError(t, err)

	rm.getter.AssertCalled(t, "Get", bpf, mock.Anything)
}

func TestRunFetchesBlueprintErrorReturnsError(t *testing.T) {
	bpf := "github.com/shipyard-run/blueprints//vault-k8s"
	rf, rm := setupRun(t)
	rf.SetArgs([]string{bpf})

	testutils.RemoveOn(&rm.getter.Mock, "Get")
	rm.getter.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	require.Error(t, err)
}

func TestRunOpensBrowserWindowForResources(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	testutils.RemoveOn(&rm.engine.Mock, "ApplyWithVariables")

	// should open
	d := &docs.Docs{ResourceBase: hcltypes.ResourceBase{Meta: hcltypes.Meta{Name: "test", Type: "docs"}}}
	d.OpenInBrowser = true

	// should open
	i := &ingress.Ingress{ResourceBase: hcltypes.ResourceBase{Meta: hcltypes.Meta{Name: "test", Type: "ingress"}}}
	i.Port = 8080
	i.OpenInBrowser = "/"

	// should open
	c := &container.Container{ResourceBase: hcltypes.ResourceBase{Meta: hcltypes.Meta{Name: "test", Type: "container"}}}
	c.Ports = []container.Port{{Host: "8080", OpenInBrowser: "https://test.container.jumppad.dev:8080"}}

	// should not be opened
	c2 := &container.Container{}
	c2.Ports = []container.Port{{OpenInBrowser: ""}}

	// should not be opened
	i2 := &ingress.Ingress{}
	i2.Port = 8080

	// should not be opened
	d2 := &docs.Docs{}

	// should be opened
	n1 := &nomad.NomadCluster{ResourceBase: hcltypes.ResourceBase{Meta: hcltypes.Meta{Name: "test", Type: "nomad_cluster"}}}
	n1.OpenInBrowser = true
	n1.APIPort = 4646

	hclconfig := hclconfig.Config{}
	hclconfig.Resources = []hcltypes.Resource{d, i, c, d2, i2, c2, n1}

	testutils.RemoveOn(&rm.engine.Mock, "ApplyWithVariables")
	rm.engine.On("ApplyWithVariables", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&hclconfig,
		nil,
	)

	err := rf.Execute()
	require.NoError(t, err)

	rm.http.AssertNumberOfCalls(t, "HealthCheckHTTP", 4)
	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 4)

	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://test.ingress.local.jmpd.in:8080/", "", map[string][]string{}, "", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "https://test.container.jumppad.dev:8080", "", map[string][]string{}, "", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://test.docs.local.jmpd.in:80", "", map[string][]string{}, "", []int{200}, 30*time.Second)
	rm.http.AssertCalled(t, "HealthCheckHTTP", "http://server.test.nomad-cluster.local.jmpd.in:4646/", "", map[string][]string{}, "", []int{200}, 30*time.Second)
}

func TestRunDoesNotOpensBrowserWindowWhenCheckError(t *testing.T) {
	rf, rm := setupRun(t)
	rf.SetArgs([]string{"/tmp"})

	testutils.RemoveOn(&rm.http.Mock, "HealthCheckHTTP")
	rm.http.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := rf.Execute()
	require.NoError(t, err)

	rm.system.AssertNumberOfCalls(t, "OpenBrowser", 0)
}
