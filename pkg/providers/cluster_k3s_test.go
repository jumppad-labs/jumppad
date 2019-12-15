package providers

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
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
   certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJWakNCL3FBREFnRUNBZ0VBTUFvR0NDcUdTTTQ5QkFNQ01DTXhJVEFmQmdOVkJBTU1HR3N6Y3kxelpYSjIKWlhJdFkyRk
FNVFUzTlRrNE1qVTNNakFlRncweE9URXlNVEF4TWpVMk1USmFGdzB5T1RFeU1EY3hNalUyTVRKYQpNQ014SVRBZkJnTlZCQU1NR0dzemN5MXpaWEoyWlhJdFkyRkFNVFUzTlRrNE1qVTNNakJaTUJNR0J5cUdTTTQ5CkFn
RUdDQ3FHU000OUF3RUhBMElBQkhSblYydVliRU53eTlROGkxd2J6ZjQ2NytGdzV2LzRBWVQ2amM4dXorM00KTmRrZEwwd0RhNGM3Y1ByOUFXM1N0ZVRYSDNtNE9mRStJYTE3L1liaDFqR2pJekFoTUE0R0ExVWREd0VCL3
dRRQpBd0lDcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUFvR0NDcUdTTTQ5QkFNQ0EwY0FNRVFDSUhFYlZwbUkzbjZwCnQrYlhKaWlFK1hiRm5XUFhtYm40OFZuNmtkYkdPM3daQWlCRDNyUjF5RjQ5R0piZmVQeXBsREdC
K3lkNVNQOEUKUmQ4OGxRWW9oRnV2enc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://127.0.0.1:64674
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
	md.On("ContainerList", mock.Anything, mock.Anything).Return("volume", nil).Once()
	md.On("ContainerList", mock.Anything, mock.Anything).Return("abc", nil).Once()
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader(kubeconfig)),
		types.ContainerPathStat{},
		nil,
	)
	md.On("CopyToContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("Running kubelet")),
		nil,
	)
	md.On("VolumeCreate", mock.Anything, mock.Anything).Return(types.Volume{Name: "hostname.volume"}, nil)

	md.On("ImageSave", mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader(kubeconfig)),
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
	params := md.Calls[3].Arguments
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
	assert.Contains(t, string(d), "server: https://127.0.0.1")

	// checks that the docker config is witten
	f, err = os.OpenFile("/tmp/.shipyard/config/hostname/kubeconfig-docker.yaml", os.O_RDONLY, 0755)
	assert.NoError(t, err)
	defer f.Close()

	d, err = ioutil.ReadAll(f)
	assert.Contains(t, string(d), "server: https://server.hostname.k3snet.shipyard")
}

func TestK3sClusterPushesLocalImages(t *testing.T) {
	cn := &config.Network{
		Name: "k3snet",
	}

	c := &config.Cluster{
		Name:       "hostname",
		Driver:     "k3s",
		Version:    "v1.0.0",
		NetworkRef: cn,
		Images: []config.Image{
			config.Image{
				Name: "myrepo/myimage:latest",
			},
		},
	}

	md, p, cleanup := setupK3sCluster(c)
	defer cleanup()

	err := p.Create()

	assert.NoError(t, err)
	md.AssertCalled(t, "VolumeCreate", mock.Anything, mock.Anything)
	md.AssertNumberOfCalls(t, "ContainerCreate", 2)
	md.AssertNumberOfCalls(t, "ImageSave", 1)
	md.AssertNumberOfCalls(t, "CopyToContainer", 1)

	params := md.Calls[1].Arguments
	vco := params[1].(volume.VolumeCreateBody)

	assert.Equal(t, "hostname.volume", vco.Name)
	// assert server properties

	// first container create will be for the image
	params = getCalls(&md.Mock, "ContainerCreate")[1].Arguments
	hc := params[2].(*container.HostConfig)

	// check the host mount has the images volume
	assert.Equal(t, "hostname.volume", hc.Mounts[0].Source)
	assert.Equal(t, "/images", hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeVolume, hc.Mounts[0].Type)
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
