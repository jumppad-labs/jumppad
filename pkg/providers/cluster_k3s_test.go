package providers

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	clients "github.com/shipyard-run/cli/pkg/clients/mocks"
	"github.com/shipyard-run/cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
)

var kubeconfig = `
kubeconfig.yam@@@@i@@@@@@@@@@@@@@@@@@@@@@2@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@lapiVersion: v1
clusters:
- cluster:
`

func setupK3sCluster(c *config.Cluster) (*clients.MockDocker, *Cluster, func()) {
	// set the shipyard env
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/tmp")

	md := &clients.MockDocker{}
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("container not found")).Once()
	md.On("ContainerList", mock.Anything, mock.Anything).Return("abc", nil).Once()
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader(kubeconfig)),
		types.ContainerPathStat{},
		nil,
	)
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("Running kubelet")),
		nil,
	)

	mk := &clients.MockKubernetes{}
	mk.Mock.On("SetConfig", mock.Anything).Return(nil)
	mk.Mock.On("GetPods").Return(
		&v1.PodList{
			Items: []v1.Pod{
				v1.Pod{
					Status: v1.PodStatus{
						Phase: "Running",
					},
				},
			},
		},
		nil,
	)

	return md, NewCluster(c, md, mk), func() {
		// cleanup
		os.Setenv("HOME", oldHome)
	}
}

func TestK3sInvalidClusterNameReturnsError(t *testing.T) {
	c := &config.Cluster{Name: "-hostname.1231", Driver: "k3s"}
	_, p, cleanup := setupK3sCluster(c)
	defer cleanup()

	err := p.Create()

	assert.True(t, xerrors.Is(err, ErrorClusterInvalidName))
}

func TestK3sReturnsErrorIfClusterExists(t *testing.T) {
	cn := &config.Network{Name: "k3snet"}
	c := &config.Cluster{Name: "hostname", Driver: "k3s", NetworkRef: cn}

	md, p, cleanup := setupK3sCluster(c)
	defer cleanup()

	removeOn(&md.Mock, "ContainerList")
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{types.Container{ID: "123sdsdsd"}}, nil)

	err := p.Create()

	md.AssertCalled(t, "ContainerList", mock.Anything, mock.Anything)

	assert.True(t, xerrors.Is(err, ErrorClusterExists))
}

func TestK3sClusterServerCreatesWithCorrectOptions(t *testing.T) {
	cn := &config.Network{
		Name: "k3snet",
	}

	c := &config.Cluster{
		Name:       "hostname",
		Driver:     "k3s",
		Version:    "v1.0.0",
		NetworkRef: cn,
	}

	md, p, cleanup := setupK3sCluster(c)
	defer cleanup()

	err := p.Create()

	assert.NoError(t, err)
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// assert server properties
	params := md.Calls[2].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)
	fqdn := params[4]

	assert.Equal(t, fmt.Sprintf("%s:%s", k3sBaseImage, c.Version), dc.Image)

	// check cluster names
	assert.Equal(t, "server.hostname.k3snet.shipyard", fqdn, "FQDN should be [nodetype].[name].[network].shipyard")
	assert.Equal(t, "server.hostname", dc.Hostname, "Hostname should be [nodetype].[hostname]")

	// check environment variables
	assert.Len(t, dc.Env, 2)
	assert.Equal(t, "K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml", dc.Env[0])
	assert.Contains(t, dc.Env[1], "K3S_CLUSTER_SECRET=")

	// check the command
	assert.Equal(t, "server", dc.Cmd[0])
	assert.Contains(t, dc.Cmd[1], "--https-listen-port=")
	assert.Equal(t, dc.Cmd[2], "--no-deploy=traefik")

	// make sure privlidged
	assert.True(t, hc.Privileged)

	// check the ports
	apiPort := strings.Split(dc.Cmd[1], "=")
	dockerPort, _ := nat.NewPort("tcp", apiPort[1])
	fmt.Println(dockerPort)

	fmt.Println(hc.PortBindings)
	fmt.Println(dc.ExposedPorts)

	assert.Len(t, dc.ExposedPorts, 1)
	assert.NotNil(t, dc.ExposedPorts[dockerPort])
	assert.Len(t, hc.PortBindings, 1)
	assert.NotNil(t, hc.PortBindings[dockerPort])

	// checks that the config is witten
	f, err := os.OpenFile("/tmp/.shipyard/config/hostname/kubeconfig.yaml", os.O_RDONLY, 0755)
	assert.NoError(t, err)
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	assert.Contains(t, string(d), "clusters")
}

// removeOn is a utility function for removing Expectations from mock objects
func removeOn(m *mock.Mock, name string) {
	ec := m.ExpectedCalls
	rc := make([]*mock.Call, 0)

	for _, c := range ec {
		if c.Method != name {
			rc = append(rc, c)
		}
	}

	m.ExpectedCalls = rc
}
