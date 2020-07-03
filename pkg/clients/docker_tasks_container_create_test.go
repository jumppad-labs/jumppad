package clients

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var wanNetwork = &config.Network{ResourceInfo: config.ResourceInfo{Name: "wan", Type: config.TypeNetwork}, Subnet: "192.168.6.0/24"}
var containerNetwork = &config.Network{ResourceInfo: config.ResourceInfo{Name: "testnet", Type: config.TypeNetwork}, Subnet: "192.168.4.0/24"}
var containerConfig = &config.Container{
	ResourceInfo: config.ResourceInfo{Name: "testcontainer", Type: config.TypeContainer},
	Image:        config.Image{Name: "consul:v1.6.1"},
	Command:      []string{"tail", "-f", "/dev/null"},
	Volumes: []config.Volume{
		config.Volume{
			Source:      "/tmp",
			Destination: "/data",
		},
	},
	Environment: []config.KV{
		config.KV{Key: "TEST", Value: "true"},
	},
	EnvVar: map[string]string{
		"key": "value",
	},
	Ports: []config.Port{
		config.Port{
			Local:    "8080",
			Host:     "9080",
			Protocol: "tcp",
		},
		config.Port{
			Local:    "8081",
			Host:     "9081",
			Protocol: "udp",
		},
	},
	PortRanges: []config.PortRange{
		config.PortRange{
			Range:      "9000-9002",
			Protocol:   "tcp",
			EnableHost: true,
		},
		config.PortRange{
			Range:      "9100-9102",
			Protocol:   "udp",
			EnableHost: false,
		},
	},
	Networks: []config.NetworkAttachment{
		config.NetworkAttachment{Name: "network.testnet"},
		config.NetworkAttachment{Name: "network.wan"},
	},
}

func createContainerConfig() (*config.Container, *config.Network, *config.Network, *clients.MockDocker, *clients.ImageLog) {
	cc := *containerConfig
	cc2 := *containerConfig
	cn := *containerNetwork
	wn := *wanNetwork

	cc2.Name = "testcontainer2"

	c := config.New()
	c.AddResource(&cc)
	c.AddResource(&cc2)
	c.AddResource(&cn)
	c.AddResource(&wn)

	mc, mic := setupContainerMocks()

	return &cc, &cn, &wn, mc, mic
}

func setupContainerMocks() (*clients.MockDocker, *clients.ImageLog) {
	md := &clients.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{ID: "test"}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("NetworkDisconnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	md.On("VolumeList", mock.Anything, mock.Anything).Return(nil, nil)
	md.On("VolumeCreate", mock.Anything, mock.Anything).Return(types.Volume{Name: "test_volume"}, nil)
	md.On("VolumeRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mic := &clients.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	return md, mic
}

func setupContainer(t *testing.T, cc *config.Container, md *clients.MockDocker, mic *clients.ImageLog) error {
	p := NewDockerTasks(md, mic, hclog.NewNullLogger())

	// create the container
	_, err := p.CreateContainer(cc)

	return err
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	// check that the docker api methods were called
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments

	cfg := params[1].(*container.Config)

	assert.Equal(t, cc.Info().Name, cfg.Hostname)
	assert.Equal(t, cc.Image.Name, cfg.Image)
	assert.Equal(t, fmt.Sprintf("%s=%s", cc.Environment[0].Key, cc.Environment[0].Value), cfg.Env[0])
	assert.Equal(t, cc.Command[0], cfg.Cmd[0])
	assert.Equal(t, cc.Command[1], cfg.Cmd[1])
	assert.Equal(t, cc.Command[2], cfg.Cmd[2])
	assert.True(t, cfg.AttachStdin)
	assert.True(t, cfg.AttachStdout)
	assert.True(t, cfg.AttachStderr)
}

func TestContainerRemovesBridgeBeforeAttachingToUserNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkDisconnect")[0].Arguments

	assert.Equal(t, "bridge", params[1])
}

func TestContainerReturnsErrorIfErrorRemovingBridge(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	removeOn(&md.Mock, "NetworkDisconnect")
	md.On("NetworkDisconnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)
}

func TestContainerAttachesToUserNetwork(t *testing.T) {
	cc, cn, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cn.Info().Name, params[1])
	assert.Equal(t, "test", params[2])
	assert.Nil(t, nc.IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerAttachesToContainerNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "container.testcontainer2"}}
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{types.Container{ID: "abc"}})

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	md.AssertNotCalled(t, "NetworkConnect")

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Equal(t, hc.NetworkMode, container.NetworkMode("container:abc"))
}

func TestContainerAttachesToContainerNetworkReturnsErrorWhenListError(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "container.testcontainer2"}}
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)
}

func TestContainerAttachesToContainerNetworkReturnsErrorWhenContainerNotFound(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "container.testcontainer2"}}
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)
}

func TestContainerRollsbackWhenUnableToConnectToNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	removeOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerDoesNOTAttachesToUserNetworkWhenNil(t *testing.T) {
	cc, nc, _, md, mic := createContainerConfig()
	cc.Networks = []config.NetworkAttachment{}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "NetworkConnect", 0)
	md.AssertNotCalled(t, "NetworkConnect", nc.Name, mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerAssignsIPToUserNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	cc.Networks[0].IPAddress = "192.168.1.123"

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].IPAddress, nc.IPAMConfig.IPv4Address)
}

func TestContainerAssignsAliasesToUserNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	cc.Networks[0].Aliases = []string{"abc", "123"}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].Aliases, nc.Aliases)
}

func TestContainerRollsbackWhenUnableToConnectToWANNetwork(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()
	removeOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom")).Once()

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerAttachesVolumeMounts(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, cc.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, cc.Volumes[0].Destination, hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeBind, hc.Mounts[0].Type)
}

func TestContainerCreatesDirectoryForVolume(t *testing.T) {
	tmpFolder := fmt.Sprintf("%s/%d", utils.ShipyardTemp(), time.Now().UnixNano())
	defer os.RemoveAll(tmpFolder)

	cc, _, _, md, mic := createContainerConfig()
	cc.Volumes[0].Source = tmpFolder

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	assert.DirExists(t, tmpFolder)
}

func TestContainerDoesNotCreatesDirectoryForVolumeWhenNotBind(t *testing.T) {
	tmpFolder := fmt.Sprintf("%s/%d", utils.ShipyardTemp(), time.Now().UnixNano())
	defer os.RemoveAll(tmpFolder)

	cc, _, _, md, mic := createContainerConfig()
	cc.Volumes[0].Source = tmpFolder
	cc.Volumes[0].Type = "volume"

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	assert.NoDirExists(t, tmpFolder)
}

func TestContainerPublishesPorts(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)

	// check the first port mapping
	exp, err := nat.NewPort(cc.Ports[0].Protocol, cc.Ports[0].Local)
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine
	assert.Equal(t, cc.Ports[0].Host, hc.PortBindings[exp][0].HostPort)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[exp][0].HostIP)

	// check the second port mapping
	exp, err = nat.NewPort(cc.Ports[1].Protocol, cc.Ports[1].Local)
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine
	assert.Equal(t, cc.Ports[1].Host, hc.PortBindings[exp][0].HostPort)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[exp][0].HostIP)
}

func TestContainerPublishesPortsRanges(t *testing.T) {
	cc, _, _, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)

	assert.Len(t, dc.ExposedPorts, 8)

	// check the first port range
	exp, err := nat.NewPort("tcp", "9000")
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	exp, err = nat.NewPort("tcp", "9001")
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	exp, err = nat.NewPort("tcp", "9002")
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine
	assert.Equal(t, "9002", hc.PortBindings[exp][0].HostPort)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[exp][0].HostIP)

	// check second range
	exp, err = nat.NewPort("udp", "9102")
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine are nil
	assert.Nil(t, hc.PortBindings[exp])
}

// removeOn is a utility function for removing Expectations from mock objects
func removeOn(m *mock.Mock, method string) {
	ec := m.ExpectedCalls
	rc := make([]*mock.Call, 0)

	for _, c := range ec {
		if c.Method != method {
			rc = append(rc, c)
		}
	}

	m.ExpectedCalls = rc
}

func getCalls(m *mock.Mock, method string) []mock.Call {
	rc := make([]mock.Call, 0)
	for _, c := range m.Calls {
		if c.Method == method {
			rc = append(rc, c)
		}
	}

	return rc
}
