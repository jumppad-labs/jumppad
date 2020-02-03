package providers

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
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
}

func createConfig() (*config.Container, *config.Network, *config.Network, *clients.MockDocker) {
	cc := *containerConfig
	cn := *containerNetwork
	wn := *wanNetwork

	cc.NetworkRef = &cn
	cc.WANRef = &wn

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
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md
}

func setupContainer(t *testing.T, cc *config.Container, md *clients.MockDocker) *Container {
	p := NewContainer(cc, md, hclog.NewNullLogger())

	// create the container
	err := p.Create()
	assert.NoError(t, err)

	// check that the docker api methods were called
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	return p
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

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
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.NotNil(t, nc.EndpointsConfig[cc.NetworkRef.Name])
	assert.Equal(t, cc.NetworkRef.Name, nc.EndpointsConfig[cc.NetworkRef.Name].NetworkID)
	assert.Nil(t, nc.EndpointsConfig[cc.NetworkRef.Name].IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerDoesNOTAttachesToUserNetworkWhenNil(t *testing.T) {
	cc, cn, _, md := createConfig()
	cc.NetworkRef = nil
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Nil(t, nc.EndpointsConfig[cn.Name])
}

func TestContainerAssignsIPToUserNetwork(t *testing.T) {
	cc, _, _, md := createConfig()
	cc.IPAddress = "192.168.1.123"
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Equal(t, cc.IPAddress, nc.EndpointsConfig[cc.NetworkRef.Name].IPAMConfig.IPv4Address)
}

func TestContainerAttachesToWANNetwork(t *testing.T) {
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.NotNil(t, nc.EndpointsConfig[cc.WANRef.Name])
	assert.Equal(t, cc.WANRef.Name, nc.EndpointsConfig[cc.WANRef.Name].NetworkID)
	assert.Nil(t, nc.EndpointsConfig[cc.WANRef.Name].IPAMConfig) // unless an IP address is set this will be nil
}

func TestContainerDoesNOTAttachesToWANNetworkWhenNil(t *testing.T) {
	cc, _, wn, md := createConfig()
	cc.WANRef = nil
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	nc := params[3].(*network.NetworkingConfig)

	assert.Nil(t, nc.EndpointsConfig[wn.Name])
}

func TestContainerAttachesVolumeMounts(t *testing.T) {
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, cc.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, cc.Volumes[0].Destination, hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeBind, hc.Mounts[0].Type)
}

func TestContainerPublishesPorts(t *testing.T) {
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

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

func TestContainerPullsImageWhenNOTCached(t *testing.T) {
	cc, _, _, md := createConfig()
	setupContainer(t, cc, md)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Image.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, types.ImageListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Image.Name), types.ImagePullOptions{})
}

func TestContainerPullsImageWithCredentialsWhenNOTCached(t *testing.T) {
	cc, _, _, md := createConfig()
	cc.Image.Username = "nicjackson"
	cc.Image.Password = "S3cur1t11"

	setupContainer(t, cc, md)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Image.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, types.ImageListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	// adding credentials to image pull
	ipo := types.ImagePullOptions{RegistryAuth: createRegistryAuth(cc.Image.Username, cc.Image.Password)}
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Image.Name), ipo)

}

func TestContainerPullsImageWithValidCredentials(t *testing.T) {
	cc, _, _, md := createConfig()
	cc.Image.Username = "nicjackson"
	cc.Image.Password = "S3cur1t11"

	setupContainer(t, cc, md)

	ipo := getCalls(&md.Mock, "ImagePull")[0].Arguments[2].(types.ImagePullOptions)

	d, err := base64.StdEncoding.DecodeString(ipo.RegistryAuth)
	assert.NoError(t, err)
	assert.Equal(t, `{"Username": "nicjackson", "Password": "S3cur1t11"}`, string(d))
}

// validate the registry auth is in the correct format
func TestContainerDoesNOTPullImageWhenCached(t *testing.T) {
	cc, _, _, md := createConfig()

	// remove the default image list which returns 0 cached images
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]types.ImageSummary{types.ImageSummary{}}, nil)

	setupContainer(t, cc, md)

	md.AssertNotCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
}
