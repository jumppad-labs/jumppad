package clients

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
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
			Source:      "/mnt/data",
			Destination: "/data",
		},
	},
	Environment: []config.KV{
		config.KV{Key: "TEST", Value: "true"},
	},
	Ports: []config.Port{
		config.Port{
			Local:    8080,
			Host:     9080,
			Protocol: "tcp",
		},
		config.Port{
			Local:    8081,
			Host:     9081,
			Protocol: "udp",
		},
	},
	Networks: []config.NetworkAttachment{
		config.NetworkAttachment{Name: "network.testnet"},
		config.NetworkAttachment{Name: "network.wan"},
	},
}

func createContainerConfig() (*config.Container, *config.Network, *config.Network, *clients.MockDocker) {
	cc := *containerConfig
	cn := *containerNetwork
	wn := *wanNetwork

	c := config.New()
	c.AddResource(&cc)
	c.AddResource(&cn)
	c.AddResource(&wn)

	return &cc, &cn, &wn, setupContainerMocks()
}

func setupContainerMocks() *clients.MockDocker {
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

	return md
}

func setupContainer(t *testing.T, cc *config.Container, md *clients.MockDocker) error {
	p := NewDockerTasks(md, hclog.NewNullLogger())

	// create the container
	_, err := p.CreateContainer(cc)

	return err
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cc, _, _, md := createContainerConfig()

	err := setupContainer(t, cc, md)
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
	cc, _, _, md := createContainerConfig()

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkDisconnect")[0].Arguments

	assert.Equal(t, "bridge", params[1])
}

func TestContainerReturnsErrorIfErrorRemovingBridge(t *testing.T) {
	cc, _, _, md := createContainerConfig()
	removeOn(&md.Mock, "NetworkDisconnect")
	md.On("NetworkDisconnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := setupContainer(t, cc, md)
	assert.Error(t, err)
}

func TestContainerAttachesToUserNetwork(t *testing.T) {
	cc, cn, _, md := createContainerConfig()

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cn.Info().Name, params[1])
	assert.Equal(t, "test", params[2])
	assert.Nil(t, nc.IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerRollsbackWhenUnableToConnectToNetwork(t *testing.T) {
	cc, _, _, md := createContainerConfig()
	removeOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := setupContainer(t, cc, md)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerDoesNOTAttachesToUserNetworkWhenNil(t *testing.T) {
	cc, nc, _, md := createContainerConfig()
	cc.Networks = []config.NetworkAttachment{}

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "NetworkConnect", 0)
	md.AssertNotCalled(t, "NetworkConnect", nc.Name, mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerAssignsIPToUserNetwork(t *testing.T) {
	cc, _, _, md := createContainerConfig()
	cc.Networks[0].IPAddress = "192.168.1.123"

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].IPAddress, nc.IPAMConfig.IPv4Address)
}

func TestContainerRollsbackWhenUnableToConnectToWANNetwork(t *testing.T) {
	cc, _, _, md := createContainerConfig()
	removeOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom")).Once()

	err := setupContainer(t, cc, md)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerAttachesVolumeMounts(t *testing.T) {
	cc, _, _, md := createContainerConfig()

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, cc.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, cc.Volumes[0].Destination, hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeBind, hc.Mounts[0].Type)
}

func TestContainerPublishesPorts(t *testing.T) {
	cc, _, _, md := createContainerConfig()

	err := setupContainer(t, cc, md)
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)

	// check the first port mapping
	exp, err := nat.NewPort(cc.Ports[0].Protocol, strconv.Itoa(cc.Ports[0].Local))
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine
	assert.Equal(t, strconv.Itoa(cc.Ports[0].Host), hc.PortBindings[exp][0].HostPort)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[exp][0].HostIP)

	// check the second port mapping
	exp, err = nat.NewPort(cc.Ports[1].Protocol, strconv.Itoa(cc.Ports[1].Local))
	assert.NoError(t, err)
	assert.NotNil(t, dc.ExposedPorts[exp])

	// check the port bindings for the local machine
	assert.Equal(t, strconv.Itoa(cc.Ports[1].Host), hc.PortBindings[exp][0].HostPort)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[exp][0].HostIP)
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
