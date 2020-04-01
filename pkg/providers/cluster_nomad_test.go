package providers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupClusterMocks sets up a happy path for mocks
func setupNomadClusterMocks() (*config.NomadCluster, *mocks.MockContainerTasks, *mocks.MockNomad, func()) {
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
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything).Return("file.tar.gz", nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("RemoveVolume", mock.Anything).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mh := &mocks.MockNomad{}
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

	return &cc, md, mh, func() {
		os.Setenv("HOME", currentHome)
		os.RemoveAll(tmpDir)
	}
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
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)

	p := NewNomadCluster(clusterNomadConfig, md, nil, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadPullsImage(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: "shipyardrun/nomad:v1.0.0"}, false)
}

func TestClusterNomadCreatesANewVolume(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateVolume", clusterConfig.Name)
}

func TestClusterNomadFailsWhenUnableToCreatesANewVolume(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	removeOn(&md.Mock, "CreateVolume")
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
	md.AssertCalled(t, "CreateVolume", clusterConfig.Name)
}

func TestClusterNomadCreatesAServer(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

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

	// validate that the custom volume has been added
	assert.Equal(t, "./files", params.Volumes[1].Source)
	assert.Equal(t, "/files", params.Volumes[1].Destination)

	// validate the API port is set
	assert.GreaterOrEqual(t, params.Ports[0].Local, 4646)
	assert.GreaterOrEqual(t, params.Ports[0].Host, 64000)
	assert.Equal(t, "tcp", params.Ports[0].Protocol)
}

func TestClusterNomadHealthChecksAPI(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create()
	assert.NoError(t, err)

	mh.AssertCalled(t, "HealthCheckAPI", mock.Anything)
}

func TestClusterNomadErrorsIfHealthFails(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	removeOn(&mh.Mock, "HealthCheckAPI")
	mh.On("HealthCheckAPI", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadImportDockerImagesPullsImages(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[0], false)
	md.AssertCalled(t, "PullImage", clusterConfig.Images[1], false)
}

func TestClusterNomadImportDockerCopiesImages(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "CopyLocalDockerImageToVolume", []string{"consul:1.6.1", "vault:1.6.1"}, "test.volume.shipyard")
}
func TestClusterNomadImportDockerCopyImageFailReturnsError(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	removeOn(&md.Mock, "CopyLocalDockerImageToVolume")
	md.On("CopyLocalDockerImageToVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestClusterNomadImportDockerRunsExecCommand(t *testing.T) {
	//TODO implement the docker import command
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	importCommand := []string{"docker", "load", "-i", "/images/file.tar.gz"}
	md.AssertCalled(t, "ExecuteCommand", "containerid", importCommand, mock.Anything)
}

func TestClusterNomadImportDockerExecFailReturnsError(t *testing.T) {
	//TODO implement the docker import command
	cc, md, mh, cleanup := setupNomadClusterMocks()
	removeOn(&md.Mock, "ExecuteCommand")
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

// Destroy Tests
func TestClusterNomadDestroyGetsIDr(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", clusterNomadConfig.Name, clusterNomadConfig.Type)
}

func TestClusterNomadDestroyWithFindIDErrorReturnsError(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.Error(t, err)
}

func TestClusterNomadDestroyWithNoIDReturns(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, nil)
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything)
}

func TestClusterNomadDestroyRemovesContainer(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", mock.Anything)
}

func TestClusterNomadDestroyRemovesVolume(t *testing.T) {
	cc, md, mh, cleanup := setupNomadClusterMocks()
	defer cleanup()

	p := NewNomadCluster(cc, md, mh, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveVolume", "test")
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
