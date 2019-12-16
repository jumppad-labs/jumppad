package providers

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	clients "github.com/shipyard-run/cli/pkg/clients/mocks"
	"github.com/shipyard-run/cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupIngress(c *config.Ingress) (*clients.MockDocker, *Ingress) {
	md := &clients.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md, &Ingress{c, md}
}

func TestCreatesIngressWithValidOptions(t *testing.T) {
	cn := &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
	cc := &config.Container{Name: "testcontainer", Image: config.Image{Name: "consul:v1.6.1"}, NetworkRef: cn, Volumes: []config.Volume{config.Volume{Source: "/mnt/data", Destination: "/data"}}}
	i := &config.Ingress{Name: "testingress", TargetRef: cc, NetworkRef: cn, Ports: []config.Port{config.Port{Protocol: "tcp", Host: 18500, Local: 8600, Remote: 8500}}}

	md, p := setupIngress(i)

	err := p.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	// second call is create
	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	name := params[4].(string)
	host := params[2].(*container.HostConfig)
	cfg := params[1].(*container.Config)
	network := params[3].(*network.NetworkingConfig)

	assert.Equal(t, "testingress.testnet.shipyard", name)

	assert.Equal(t, "testingress", cfg.Hostname)
	assert.Equal(t, "shipyardrun/ingress:latest", cfg.Image)

	assert.Equal(t, "--service-name", cfg.Cmd[0])
	assert.Equal(t, "testcontainer.testnet.shipyard", cfg.Cmd[1])

	assert.Equal(t, "--ports", cfg.Cmd[2])
	assert.Equal(t, "8600:8500", cfg.Cmd[3])

	// check the ports
	dockerPort, _ := nat.NewPort("tcp", "8600")
	assert.NotNil(t, cfg.ExposedPorts[dockerPort])
	assert.NotNil(t, host.PortBindings[dockerPort])
	assert.Equal(t, "18500", host.PortBindings[dockerPort][0].HostPort)

	// check the network
	assert.NotNil(t, network.EndpointsConfig[cn.Name])
}

func TestCreatesIngressWithContainerOptions(t *testing.T) {
	cn := &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
	cc := &config.Container{Name: "testcontainer", Image: config.Image{Name: "consul:v1.6.1"}, NetworkRef: cn, Volumes: []config.Volume{config.Volume{Source: "/mnt/data", Destination: "/data"}}}
	i := &config.Ingress{Name: "testingress", TargetRef: cc, NetworkRef: cn, Ports: []config.Port{config.Port{Protocol: "tcp", Host: 18500, Local: 8600, Remote: 8500}}}

	md, p := setupIngress(i)

	err := p.Create()
	assert.NoError(t, err)

	// second call is create
	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	name := params[4].(string)
	cfg := params[1].(*container.Config)
	network := params[3].(*network.NetworkingConfig)

	assert.Equal(t, "testingress.testnet.shipyard", name)

	assert.Equal(t, "--service-name", cfg.Cmd[0])
	assert.Equal(t, "testcontainer.testnet.shipyard", cfg.Cmd[1])

	// check the network
	assert.NotNil(t, network.EndpointsConfig[cn.Name])
}

func TestCreatesIngressWithK8sClusterOptions(t *testing.T) {
	cn := &config.Network{Name: "testnet", Subnet: "192.168.4.0/24"}
	cc := &config.Cluster{Name: "testcontainer", Driver: "k3s", NetworkRef: cn}
	i := &config.Ingress{Name: "testingress", Service: "svc/consul-consul", TargetRef: cc, NetworkRef: cn, Ports: []config.Port{config.Port{Protocol: "tcp", Host: 18500, Local: 8600, Remote: 8500}}}

	md, p := setupIngress(i)

	err := p.Create()
	assert.NoError(t, err)

	// second call is create
	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	name := params[4].(string)
	cfg := params[1].(*container.Config)
	host := params[2].(*container.HostConfig)
	network := params[3].(*network.NetworkingConfig)

	assert.Equal(t, "testingress.testnet.shipyard", name)

	assert.Equal(t, "--proxy-type", cfg.Cmd[0])
	assert.Equal(t, "kubernetes", cfg.Cmd[1])

	assert.Equal(t, "--service-name", cfg.Cmd[2])
	assert.Equal(t, i.Service, cfg.Cmd[3])

	// check the network
	assert.NotNil(t, network.EndpointsConfig[cn.Name])

	// check mounts the kubeconfig
	assert.Equal(t, "/.kube/kubeconfig.yml", host.Mounts[0].Target)
	assert.Contains(t, host.Mounts[0].Source, ".shipyard/config/testcontainer/kubeconfig-docker.yaml")
	assert.Equal(t, "KUBECONFIG=/.kube/kubeconfig.yml", cfg.Env[0])
}
