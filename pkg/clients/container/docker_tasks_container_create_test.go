package container

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
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	dtypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/images"
	imocks "github.com/jumppad-labs/jumppad/pkg/clients/images/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/mohae/deepcopy"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

var containerConfig = &dtypes.Container{
	Name:    "testcontainer",
	Image:   &dtypes.Image{Name: "consul:v1.6.1"},
	Command: []string{"tail", "-f", "/dev/null"},
	Volumes: []dtypes.Volume{
		dtypes.Volume{
			Source:                      "/tmp",
			Destination:                 "/data",
			ReadOnly:                    false,
			BindPropagation:             "shared",
			BindPropagationNonRecursive: true,
		},
	},
	Environment: map[string]string{
		"key": "value",
	},
	Ports: []dtypes.Port{
		dtypes.Port{
			Local:    "8080",
			Host:     "9080",
			Protocol: "tcp",
		},
		dtypes.Port{
			Local:    "8081",
			Host:     "9081",
			Protocol: "udp",
		},
	},
	PortRanges: []dtypes.PortRange{
		dtypes.PortRange{
			Range:      "9000-9002",
			Protocol:   "tcp",
			EnableHost: true,
		},
		dtypes.PortRange{
			Range:      "9100-9102",
			Protocol:   "udp",
			EnableHost: false,
		},
	},
	Resources: &dtypes.Resources{
		CPU:    1000,
		Memory: 1000,
		CPUPin: []int{1, 4},
	},
	Networks: []dtypes.NetworkAttachment{
		dtypes.NetworkAttachment{ID: "network.testnet"},
		dtypes.NetworkAttachment{ID: "network.wan"},
	},
}

func createContainerConfig() (*dtypes.Container, *mocks.Docker, *imocks.ImageLog) {
	cc := deepcopy.Copy(containerConfig).(*dtypes.Container)
	cc2 := deepcopy.Copy(containerConfig).(*dtypes.Container)

	cc2.Name = "testcontainer2"

	mc, mic := setupContainerMocks()

	return cc, mc, mic
}

func setupContainerMocks() (*mocks.Docker, *imocks.ImageLog) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ContainerInspect", mock.Anything, mock.Anything).Return(types.ContainerJSON{NetworkSettings: &types.NetworkSettings{Networks: map[string]*network.EndpointSettings{"bridge": nil}}}, nil)
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.CreateResponse{ID: "test"}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("NetworkDisconnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("NetworkList", mock.Anything, mock.Anything).Return(
		[]types.NetworkResource{
			types.NetworkResource{ID: "abc", Labels: map[string]string{"id": "network.testnet"}, IPAM: network.IPAM{Config: []network.IPAMConfig{{Subnet: "10.0.0.0/24"}}}},
			types.NetworkResource{ID: "123", Labels: map[string]string{"id": "network.wan"}, IPAM: network.IPAM{Config: []network.IPAMConfig{{Subnet: "10.2.0.0/24"}}}},
		}, nil)

	md.On("VolumeList", mock.Anything, mock.Anything).Return(volume.ListResponse{}, nil)
	md.On("VolumeCreate", mock.Anything, mock.Anything).Return(volume.Volume{Name: "test_volume"}, nil)
	md.On("VolumeRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &imocks.ImageLog{}
	mic.On("Log", mock.Anything, mock.Anything).Return(nil)

	return md, mic
}

func setupContainer(t *testing.T, cc *dtypes.Container, md *mocks.Docker, mic images.ImageLog) error {
	p, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	// create the container
	_, err := p.CreateContainer(cc)

	return err
}

func TestContainerCreatesCorrectly(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	// check that the docker api methods were called
	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments

	cfg := params[1].(*container.Config)

	assert.Equal(t, cc.Name, cfg.Hostname)
	assert.Equal(t, cc.Image.Name, cfg.Image)
	assert.Equal(t, fmt.Sprintf("key=%s", cc.Environment["key"]), cfg.Env[0])
	assert.Equal(t, cc.Command[0], cfg.Cmd[0])
	assert.Equal(t, cc.Command[1], cfg.Cmd[1])
	assert.Equal(t, cc.Command[2], cfg.Cmd[2])
	assert.True(t, cfg.AttachStdin)
	assert.True(t, cfg.AttachStdout)
	assert.True(t, cfg.AttachStderr)
}

func TestContainerRemovesBridgeBeforeAttachingToUserNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "NetworkDisconnect")[0].Arguments

	assert.Equal(t, "bridge", params[1])
}

func TestContainerAttachesToUserNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].Name, params[1])
	assert.Equal(t, "test", params[2])
	assert.Nil(t, nc.IPAMConfig) // unless an IP address is set this will be nil
}

func TestSidecarContainerAttachesToContainerNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks = []dtypes.NetworkAttachment{dtypes.NetworkAttachment{ID: "abc", Name: "container.testcontainer2", IsContainer: true}}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	md.AssertNotCalled(t, "NetworkConnect")

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Equal(t, hc.NetworkMode, container.NetworkMode("container:abc"))
}

func TestContainerAttachesToContainerNetworkReturnsErrorWhenListError(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks = []dtypes.NetworkAttachment{dtypes.NetworkAttachment{Name: "container.testcontainer2"}}
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)
}

func TestContainerAttachesToContainerNetworkReturnsErrorWhenContainerNotFound(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks = []dtypes.NetworkAttachment{dtypes.NetworkAttachment{Name: "container.testcontainer2"}}
	md.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)
}

func TestContainerRollsbackWhenUnableToConnectToNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()
	testutils.RemoveOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerDoesNOTAttachesToUserNetworkWhenNil(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks = []dtypes.NetworkAttachment{}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "NetworkConnect", 0)
}

func TestContainerAssignsIPToUserNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks[0].IPAddress = "192.168.1.123"

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].IPAddress, nc.IPAMConfig.IPv4Address)
}

func TestContainerAssignsAliasesToUserNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Networks[0].Aliases = []string{"abc", "123"}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "NetworkConnect")[0].Arguments
	nc := params[3].(*network.EndpointSettings)

	assert.Equal(t, cc.Networks[0].Aliases, nc.Aliases)
}

func TestContainerRollsbackWhenUnableToConnectToWANNetwork(t *testing.T) {
	cc, md, mic := createContainerConfig()
	testutils.RemoveOn(&md.Mock, "NetworkConnect")
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	md.On("NetworkConnect", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom")).Once()

	err := setupContainer(t, cc, md, mic)
	assert.Error(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, mock.Anything, mock.Anything)
}

func TestContainerAttachesVolumeMounts(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Equal(t, cc.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, cc.Volumes[0].Destination, hc.Mounts[0].Target)
	assert.Equal(t, mount.TypeBind, hc.Mounts[0].Type)
	assert.Equal(t, mount.PropagationShared, hc.Mounts[0].BindOptions.Propagation)
	assert.True(t, hc.Mounts[0].BindOptions.NonRecursive)
}

func TestContainerIgnoresBindOptionsForVolumesTypeVolume(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Volumes[0].Type = "volume"

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.Len(t, hc.Binds, 0)
}

func TestContainerSetsReadOnlyForVolumeTypeVolume(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Volumes[0].Type = "volume"
	cc.Volumes[0].ReadOnly = true

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)

	assert.Len(t, hc.Mounts, 1)
	assert.True(t, hc.Mounts[0].ReadOnly)
}

func TestContainerSetsBindOptionsForVolumeTypeBind(t *testing.T) {
	tt := map[string]mount.Propagation{
		"":         mount.PropagationRPrivate,
		"shared":   mount.PropagationShared,
		"slave":    mount.PropagationSlave,
		"private":  mount.PropagationPrivate,
		"rslave":   mount.PropagationRSlave,
		"rprivate": mount.PropagationRPrivate,
	}

	for k, v := range tt {
		cc, md, mic := createContainerConfig()
		cc.Volumes[0].BindPropagation = k

		err := setupContainer(t, cc, md, mic)
		assert.NoError(t, err)

		params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
		hc := params[2].(*container.HostConfig)

		assert.Equal(t, v, hc.Mounts[0].BindOptions.Propagation)
	}
}

func TestContainerCreatesDirectoryForVolume(t *testing.T) {
	tmpFolder := fmt.Sprintf("%s/%d", utils.ShipyardTemp(), time.Now().UnixNano())
	defer os.RemoveAll(tmpFolder)

	cc, md, mic := createContainerConfig()
	cc.Volumes[0].Source = tmpFolder

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	assert.DirExists(t, tmpFolder)
}

func TestContainerDoesNotCreatesDirectoryForVolumeWhenNotBind(t *testing.T) {
	tmpFolder := fmt.Sprintf("%s/%d", utils.ShipyardTemp(), time.Now().UnixNano())
	defer os.RemoveAll(tmpFolder)

	cc, md, mic := createContainerConfig()
	cc.Volumes[0].Source = tmpFolder
	cc.Volumes[0].Type = "volume"

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	assert.NoDirExists(t, tmpFolder)
}

func TestContainerPublishesPorts(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
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
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
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

func TestContainerConfiguresResources(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)
	assert.NotEmpty(t, hc.Resources)

	assert.Equal(t, hc.Resources.Memory, int64(1000000000))
	assert.Equal(t, hc.Resources.CPUQuota, int64(100000))
	assert.Equal(t, hc.Resources.CpusetCpus, "1,4")
}

func TestContainerConfiguresRetryWhenCountGreater0(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.MaxRestartCount = 10

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)
	assert.NotEmpty(t, hc.Resources)

	assert.Equal(t, hc.RestartPolicy.MaximumRetryCount, 10)
	assert.Equal(t, hc.RestartPolicy.Name, "on-failure")
}

func TestContainerNotConfiguresRetryWhen0(t *testing.T) {
	cc, md, mic := createContainerConfig()

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	hc := params[2].(*container.HostConfig)
	assert.NotEmpty(t, hc.Resources)

	assert.Equal(t, hc.RestartPolicy.MaximumRetryCount, 0)
}

func TestContainerAddUserWhenSpecified(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.RunAs = &dtypes.User{
		User:  "1010",
		Group: "1011",
	}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[1].(*container.Config)
	assert.Equal(t, "1010:1011", dc.User)
}

func TestContainerAddCapabilities(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Capabilities = &dtypes.Capabilities{Add: []string{"SYS_ADMIN", "SYS_CHROOT"}}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[2].(*container.HostConfig)
	assert.Equal(t, "SYS_ADMIN", dc.CapAdd[0])
	assert.Equal(t, "SYS_CHROOT", dc.CapAdd[1])
}

func TestContainerDropCapabilities(t *testing.T) {
	cc, md, mic := createContainerConfig()
	cc.Capabilities = &dtypes.Capabilities{Drop: []string{"SYS_ADMIN", "SYS_CHROOT"}}

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[2].(*container.HostConfig)
	assert.Equal(t, "SYS_ADMIN", dc.CapDrop[0])
	assert.Equal(t, "SYS_CHROOT", dc.CapDrop[1])
}
