package providers

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var wanNetwork = &config.Network{Name: "wan", Subnet: "192.168.6.0/24"}
var containerNetwork = &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
var containerConfig = &config.Container{
	Name:    "testcontainer",
	Image:   config.Image{Name: "consul:v1.6.1"},
	Command: []string{"tail", "-f", "/dev/null"},
	Volumes: []config.Volume{
		config.Volume{
			Source:      "/mnt/data",
			Destination: "/data",
		},
	},
	Environment: []config.KV{
		config.KV{Key: "TEST", Value: "true"},
	},
}

func createConfig() (*config.Container, *config.Network, *config.Network) {
	cc := *containerConfig
	cn := *containerNetwork
	wn := *wanNetwork

	cc.NetworkRef = &cn
	cc.WANRef = &wn

	return &cc, &cn, &wn
}

func setupContainerMocks(t *testing.T, cc *config.Container) *clients.MockDocker {
	md := &clients.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md
}

func setupContainer(t *testing.T, cc *config.Container) (*Container, *clients.MockDocker) {
	md := setupContainerMocks(t, cc)
	p := NewContainer(cc, md, hclog.NewNullLogger())

	// create the container
	err := p.Create()
	assert.NoError(t, err)

	// check that the docker api methods were called
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	return p, md
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cc, _, _ := createConfig()
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments

	cfg := params[1].(*container.Config)

	assert.Equal(t, cc.Name, cfg.Hostname)
	assert.Equal(t, cc.Image.Name, cfg.Image)
	assert.Equal(t, fmt.Sprintf("%s=%s", cc.Environment[0].Key, cc.Environment[0].Value), cfg.Env[0])
	assert.Equal(t, cc.Command[0], cfg.Cmd[0])
	assert.Equal(t, cc.Command[1], cfg.Cmd[1])
	assert.Equal(t, cc.Command[2], cfg.Cmd[2])
	assert.True(t, cfg.AttachStdin)
	assert.True(t, cfg.AttachStdout)
	assert.True(t, cfg.AttachStderr)
}

func TestContainerAttachesToUserNetwork(t *testing.T) {
	cc, _, _ := createConfig()
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.NotNil(t, nc.EndpointsConfig[cc.NetworkRef.Name])
	assert.Equal(t, cc.NetworkRef.Name, nc.EndpointsConfig[cc.NetworkRef.Name].NetworkID)
	assert.Nil(t, nc.EndpointsConfig[cc.NetworkRef.Name].IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerDoesNOTAttachesToUserNetworkWhenNil(t *testing.T) {
	cc, cn, _ := createConfig()
	cc.NetworkRef = nil
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Nil(t, nc.EndpointsConfig[cn.Name])
}

func TestContainerAssignsIPToUserNetwork(t *testing.T) {
	cc, _, _ := createConfig()
	cc.IPAddress = "192.168.1.123"
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Equal(t, cc.IPAddress, nc.EndpointsConfig[cc.NetworkRef.Name].IPAMConfig.IPv4Address)
}

func TestContainerAttachesToWANNetwork(t *testing.T) {
	cc, _, _ := createConfig()
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.NotNil(t, nc.EndpointsConfig[cc.WANRef.Name])
	assert.Equal(t, cc.WANRef.Name, nc.EndpointsConfig[cc.WANRef.Name].NetworkID)
	assert.Nil(t, nc.EndpointsConfig[cc.WANRef.Name].IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerDoesNOTAttachesToWANNetworkWhenNil(t *testing.T) {
	cc, _, wn := createConfig()
	cc.WANRef = nil
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Nil(t, nc.EndpointsConfig[wn.Name])
}

func TestContainerAttachesVolumeMounts(t *testing.T) {
	cc, _, _ := createConfig()
	_, md := setupContainer(t, cc)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, cc.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, cc.Volumes[0].Destination, hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeBind, hc.Mounts[0].Type)
}

func TestContainerPublishesPorts(t *testing.T) {
	/*
		cc, _, _ := createConfig()
		_, md := setupContainer(t, cc)

		params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
		dc := params[1].(*container.Config)
		hc := params[3].(*container.HostConfig)
	*/

	t.SkipNow()
}

func TestContainerPullsImageWhenNOTCached(t *testing.T) {
	t.SkipNow()
}

func TestContainerDoesNOTPullImageWhenCached(t *testing.T) {
	t.SkipNow()
}
