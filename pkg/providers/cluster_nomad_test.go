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
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

// setupClusterMocks sets up a happy path for mocks
func setupNomadClusterMocks(t *testing.T) (*config.NomadCluster, *mocks.MockContainerTasks, *mocks.MockNomad) {
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
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything, mock.Anything).Return([]string{"file.tar.gz"}, nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("RemoveVolume", mock.Anything).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mh := &mocks.MockNomad{}
	mh.On("SetConfig", mock.Anything, mock.Anything).Return(nil)
	mh.On("HealthCheckAPI", mock.Anything).Return(nil)

	// set the home folder to a temp folder
	tmpDir, _ := ioutil.TempDir("", "")
	currentHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// copy the config
	cc := *clusterNomadConfig
	cn := *clusterNetwork

	c := config.New()
	c.AddResource(&cc)
	c.AddResource(&cn)

	t.Cleanup(func() {
		os.Setenv("HOME", currentHome)
		os.RemoveAll(tmpDir)
	})

	return &cc, md, mh
}

func TestClusterNomadErrorsWhenUnableToLookupIDs(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := NewNomadCluster(clusterNomadConfig, md, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadErrorsWhenClusterExists(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", "server."+clusterNomadConfig.Name, mock.Anything).Return([]string{"abc"}, nil)

	p := NewNomadCluster(clusterNomadConfig, md, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadErrorsWhenWorkerNodesExist(t *testing.T) {
	cc, md, _ := setupNomadClusterMocks(t)
	cc.ClientNodes = 3
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", "1.client."+clusterNomadConfig.Name, mock.Anything).Return([]string{"abc"}, nil)

	p := NewNomadCluster(cc, md, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadPullsImage(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: "shipyardrun/nomad:v1.0.0"}, false)
}

func TestClusterNomadPullsImageWithDefault(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.Version = "" // reset the version

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: "shipyardrun/nomad:" + nomadBaseVersion}, false)
}

func TestClusterNomadCreatesANewVolume(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateVolume", utils.ImageVolumeName)
}

func TestClusterNomadFailsWhenUnableToCreatesANewVolume(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	removeOn(&md.Mock, "CreateVolume")
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
	md.AssertCalled(t, "CreateVolume", utils.ImageVolumeName)
}

func TestClusterNomadCreatesAServer(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	cc.Volumes = []config.Volume{config.Volume{Source: "./files", Destination: "/files"}}

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// validate the basic details for the server container
	assert.Contains(t, params.Name, "server")
	assert.Contains(t, params.Image.Name, "nomad")
	assert.Equal(t, clusterNetwork.Name, params.Networks[0].Name)
	assert.True(t, params.Privileged)

	// validate that the volume is correctly set
	assert.Equal(t, "123", params.Volumes[0].Source)
	assert.Equal(t, "/images", params.Volumes[0].Destination)
	assert.Equal(t, "volume", params.Volumes[0].Type)

	// validate that the config volume has been added
	assert.Contains(t, params.Volumes[1].Source, "test/server_config.hcl")
	assert.Equal(t, "/etc/nomad.d/config.hcl", params.Volumes[1].Destination)

	// validate that the custom volume has been added
	assert.Equal(t, "./files", params.Volumes[2].Source)
	assert.Equal(t, "/files", params.Volumes[2].Destination)

	// validate the API port is set
	intLocal, _ := strconv.Atoi(params.Ports[0].Local)
	intHost, _ := strconv.Atoi(params.Ports[0].Host)
	assert.GreaterOrEqual(t, intLocal, 4646)
	assert.GreaterOrEqual(t, intHost, 64000)
	assert.Equal(t, "tcp", params.Ports[0].Protocol)

	// validate the Connector port is set
	intLocal, _ = strconv.Atoi(params.Ports[1].Local)
	intHost, _ = strconv.Atoi(params.Ports[1].Host)
	assert.GreaterOrEqual(t, intLocal, 19090)
	assert.GreaterOrEqual(t, intHost, 64000)
	assert.Equal(t, "tcp", params.Ports[0].Protocol)
}

func TestClusterNomadCreatesClientNodes(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.ClientNodes = 3

	cc.Volumes = []config.Volume{config.Volume{Source: "./files", Destination: "/files"}}

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "CreateContainer", 4)
}

func TestClusterNomadCreatesClientNodesWithCorrectDetails(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.ClientNodes = 1

	cc.Volumes = []config.Volume{config.Volume{Source: "./files", Destination: "/files"}}

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "CreateContainer", 2)

	params := getCalls(&md.Mock, "CreateContainer")[1].Arguments[0].(*config.Container)

	// validate the basic details for the server container
	assert.Contains(t, params.Name, "1.client.test")
	assert.Contains(t, params.Image.Name, "nomad")
	assert.Equal(t, clusterNetwork.Name, params.Networks[0].Name)
	assert.True(t, params.Privileged)

	// validate that the volume is correctly set
	assert.Equal(t, "123", params.Volumes[0].Source)
	assert.Equal(t, "/images", params.Volumes[0].Destination)
	assert.Equal(t, "volume", params.Volumes[0].Type)

	// validate that the config volume has been added
	assert.Contains(t, params.Volumes[1].Source, "test/client_config.hcl")
	assert.Equal(t, "/etc/nomad.d/config.hcl", params.Volumes[1].Destination)

	// validate that the custom volume has been added
	assert.Equal(t, "./files", params.Volumes[2].Source)
	assert.Equal(t, "/files", params.Volumes[2].Destination)
}

func TestClusterNomadGeneratesConfig(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	_, configPath := utils.CreateClusterConfigPath(cc.Name)
	assert.FileExists(t, configPath)
}

func TestClusterNomadHealthChecksAPI(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create()
	assert.NoError(t, err)

	mh.AssertCalled(t, "HealthCheckAPI", mock.Anything)
}

func TestClusterNomadErrorsIfHealthFails(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	removeOn(&mh.Mock, "HealthCheckAPI")
	mh.On("HealthCheckAPI", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadImportDockerImagesPullsImages(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[0], false)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[1], false)
}

func TestClusterNomadImportDockerCopiesImages(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CopyLocalDockerImageToVolume", []string{"consul:1.6.1", "vault:1.6.1"}, "images.volume.shipyard.run", false)
}

func TestClusterNomadImportDockerCopyImageFailReturnsError(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	removeOn(&md.Mock, "CopyLocalDockerImageToVolume")
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadImportDockerRunsExecCommand(t *testing.T) {
	//TODO implement the docker import command
	cc, md, mh := setupNomadClusterMocks(t)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	importCommand := []string{"docker", "load", "-i", "/images/file.tar.gz"}
	md.AssertCalled(t, "ExecuteCommand", "containerid", importCommand, mock.Anything, mock.Anything, mock.Anything)
}

func TestClusterNomadImportDockerExecFailReturnsError(t *testing.T) {
	//TODO implement the docker import command
	cc, md, mh := setupNomadClusterMocks(t)
	removeOn(&md.Mock, "ExecuteCommand")
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

// Destroy Tests
func TestClusterNomadDestroyGetsIDs(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.ClientNodes = 3

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", "server."+clusterNomadConfig.Name, clusterNomadConfig.Type)
	md.AssertCalled(t, "FindContainerIDs", "1.client."+clusterNomadConfig.Name, clusterNomadConfig.Type)
	md.AssertCalled(t, "FindContainerIDs", "2.client."+clusterNomadConfig.Name, clusterNomadConfig.Type)
	md.AssertCalled(t, "FindContainerIDs", "3.client."+clusterNomadConfig.Name, clusterNomadConfig.Type)
}

func TestClusterNomadDestroyWithFindIDErrorReturnsError(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", "server."+clusterNomadConfig.Name, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.Error(t, err)
}

func TestClusterNomadDestroyWithFindIDClientNodeErrorReturnsError(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.ClientNodes = 1
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", "server."+clusterNomadConfig.Name, mock.Anything).Return(nil, nil)
	md.On("FindContainerIDs", "1.client."+clusterNomadConfig.Name, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.Error(t, err)
}

func TestClusterNomadDestroyWithNoIDReturns(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, nil)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything)
}

func TestClusterNomadDestroyRemovesContainer(t *testing.T) {
	cc, md, mh := setupNomadClusterMocks(t)
	cc.ClientNodes = 3
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertNumberOfCalls(t, "RemoveContainer", 4)
}

var clusterNomadConfig = &config.NomadCluster{
	ResourceInfo: config.ResourceInfo{Name: "test", Type: config.TypeNomadCluster},
	Version:      "v1.0.0",
	Images: []config.Image{
		config.Image{Name: "consul:1.6.1"},
		config.Image{Name: "vault:1.6.1"},
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
}
