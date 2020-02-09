package providers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupClusterMocks sets up a happy path for mocks
func setupClusterMocks() (config.Cluster, *mocks.MockContainerTasks, *mocks.MockKubernetes, func()) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil)
	md.On("PullImage", mock.Anything, mock.Anything).Return(nil)
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("123", nil)
	md.On("CreateContainer", mock.Anything).Return("containerid", nil)
	md.On("ContainerLogs", mock.Anything, true, true).Return(
		ioutil.NopCloser(bytes.NewBufferString("Running kubelet")),
		nil,
	)
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything).Return("/images/file.tar.gz", nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("RemoveVolume", mock.Anything).Return(nil)

	// set the home folder to a temp folder
	tmpDir, _ := ioutil.TempDir("", "")
	currentHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// write the kubeconfi
	_, destPath, _ := utils.CreateKubeConfigPath(clusterConfig.Name)
	kcf, err := os.Create(destPath)
	if err != nil {
		panic(err)
	}
	kcf.WriteString(kubeconfig)
	kcf.Close()

	// create the Kubernetes client mock
	mk := &mocks.MockKubernetes{}
	mk.Mock.On("SetConfig", mock.Anything).Return(nil)
	mk.Mock.On("HealthCheckPods", mock.Anything, mock.Anything).Return(nil)

	// copy the config
	cc := clusterConfig
	cn := clusterNetwork
	cc.NetworkRef = &cn

	return cc, md, mk, func() {
		os.Setenv("HOME", currentHome)
		os.RemoveAll(tmpDir)
	}
}

func TestClusterK3ErrorsWhenUnableToLookupIDs(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	mk := &mocks.MockKubernetes{}
	p := NewCluster(clusterConfig, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3ErrorsWhenClusterExists(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)

	mk := &mocks.MockKubernetes{}
	p := NewCluster(clusterConfig, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3PullsImage(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: "rancher/k3s:v1.0.0"}, false)
}

func TestClusterK3CreatesANewVolume(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateVolume", clusterConfig.Name)
}

func TestClusterK3FailsWhenUnableToCreatesANewVolume(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	removeOn(&md.Mock, "CreateVolume")
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
	md.AssertCalled(t, "CreateVolume", clusterConfig.Name)
}

func TestClusterK3CreatesAServer(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)

	// validate the basic details for the server container
	assert.Contains(t, params.Name, "server")
	assert.Contains(t, params.Image.Name, "rancher")
	assert.Equal(t, &clusterNetwork, params.NetworkRef)
	assert.True(t, params.Privileged)

	// validate that the volume is correctly set
	assert.Equal(t, "123", params.Volumes[0].Source)
	assert.Equal(t, "/images", params.Volumes[0].Destination)
	assert.Equal(t, "volume", params.Volumes[0].Type)

	// validate the API port is set
	assert.GreaterOrEqual(t, params.Ports[0].Local, 64000)
	assert.GreaterOrEqual(t, params.Ports[0].Local, params.Ports[0].Host)
	assert.Equal(t, "tcp", params.Ports[0].Protocol)

	// validate the command
	assert.Equal(t, "server", params.Command[0])
	assert.Contains(t, params.Command[1], strconv.Itoa(params.Ports[0].Local))
	assert.Contains(t, params.Command[2], "traefik")
}

func TestClusterK3sErrorsIfServerNOTStart(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	removeOn(&md.Mock, "ContainerLogs")
	md.On("ContainerLogs", mock.Anything, true, true).Return(
		ioutil.NopCloser(bytes.NewBufferString("Not running")),
		nil,
	)

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3sDownloadsConfig(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	_, destPath, _ := utils.CreateKubeConfigPath(clusterConfig.Name)
	params := getCalls(&md.Mock, "CopyFromContainer")[0].Arguments
	assert.Equal(t, "containerid", params.String(0))
	assert.Equal(t, "/output/kubeconfig.yaml", params.String(1))
	assert.Equal(t, destPath, params.String(2))
}

func TestClusterK3sRaisesErrorWhenUnableToDownloadConfig(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	removeOn(&md.Mock, "CopyFromContainer")
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3sCreatesDockerConfig(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	// check the kubeconfig file for docker uses a network ip not localhost

	// check file has been written
	_, _, dockerPath := utils.CreateKubeConfigPath(clusterConfig.Name)
	f, err := os.Open(dockerPath)
	assert.NoError(t, err)
	defer f.Close()

	// check file contains docker ip
	d, err := ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.Contains(t, string(d), fmt.Sprintf("server.%s", utils.FQDN(clusterConfig.Name, clusterConfig.NetworkRef.Name)))
}

func TestClusterK3sCreatesKubeClient(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	mk.AssertCalled(t, "SetConfig", mock.Anything)
}

func TestClusterK3sErrorsWhenFailedToCreateKubeClient(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	removeOn(&mk.Mock, "SetConfig")
	mk.Mock.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3sWaitsForPods(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	mk.AssertCalled(t, "HealthCheckPods", []string{""}, startTimeout)
}

func TestClusterK3sErrorsWhenWaitsForPodsFail(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()
	removeOn(&mk.Mock, "HealthCheckPods")
	mk.On("HealthCheckPods", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3sImportDockerImagesPullsImages(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[0], false)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[1], false)
}

func TestClusterK3sImportDockerCopiesImages(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CopyLocalDockerImageToVolume", []string{"consul:1.6.1", "vault:1.6.1"}, "test.volume.shipyard")
}
func TestClusterK3sImportDockerCopyImageFailReturnsError(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	removeOn(&md.Mock, "CopyLocalDockerImageToVolume")
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterK3sImportDockerRunsExecCommand(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "ExecuteCommand", "containerid", mock.Anything, mock.Anything)
}

func TestClusterK3sImportDockerExecFailReturnsError(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	removeOn(&md.Mock, "ExecuteCommand")
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

// Destroy Tests
func TestClusterK3sDestroyGetsIDr(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", clusterConfig.Name, clusterConfig.NetworkRef.Name)
}

func TestClusterK3sDestroyWithFindIDErrorReturnsError(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Destroy()
	assert.Error(t, err)
}

func TestClusterK3sDestroyWithNoIDReturns(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, nil)
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything)
}

func TestClusterK3sDestroyRemovesContainer(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", mock.Anything)
}

func TestClusterK3sDestroyRemovesVolume(t *testing.T) {
	cc, md, mk, cleanup := setupClusterMocks()
	defer cleanup()

	p := NewCluster(cc, md, mk, nil, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveVolume", "test")
}

var clusterNetwork = config.Network{Name: "cloud"}

var clusterConfig = config.Cluster{
	Name:       "test",
	Driver:     "k3s",
	Version:    "v1.0.0",
	NetworkRef: &clusterNetwork,
	Images: []config.Image{
		config.Image{Name: "consul:1.6.1"},
		config.Image{Name: "vault:1.6.1"},
	},
}

var kubeconfig = `
apiVersion: v1
clusters:
- cluster:
   certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJWakNCL3FBREFnRUNBZ0VBTUFvR0NDcUdTTTQ5QkFNQ01DTXhJVEFmQmdOVkJBTU1HR3N6Y3kxelpYSjIKWlhJdFkyRk
FNVFUzTlRrNE1qVTNNakFlRncweE9URXlNVEF4TWpVMk1USmFGdzB5T1RFeU1EY3hNalUyTVRKYQpNQ014SVRBZkJnTlZCQU1NR0dzemN5MXpaWEoyWlhJdFkyRkFNVFUzTlRrNE1qVTNNakJaTUJNR0J5cUdTTTQ5CkFn
RUdDQ3FHU000OUF3RUhBMElBQkhSblYydVliRU53eTlROGkxd2J6ZjQ2NytGdzV2LzRBWVQ2amM4dXorM00KTmRrZEwwd0RhNGM3Y1ByOUFXM1N0ZVRYSDNtNE9mRStJYTE3L1liaDFqR2pJekFoTUE0R0ExVWREd0VCL3
dRRQpBd0lDcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUFvR0NDcUdTTTQ5QkFNQ0EwY0FNRVFDSUhFYlZwbUkzbjZwCnQrYlhKaWlFK1hiRm5XUFhtYm40OFZuNmtkYkdPM3daQWlCRDNyUjF5RjQ5R0piZmVQeXBsREdC
K3lkNVNQOEUKUmQ4OGxRWW9oRnV2enc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://127.0.0.1:64674
`
