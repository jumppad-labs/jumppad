package cache

import (
	"encoding/json"
	"path/filepath"
	"testing"

	dtypes "github.com/docker/docker/api/types"
	htypes "github.com/jumppad-labs/hclconfig/types"
	cmocks "github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupImageCacheTests(t *testing.T) (*ImageCache, *cmocks.ContainerTasks) {
	cc := &ImageCache{ResourceMetadata: htypes.ResourceMetadata{Name: "test"}}

	md := &cmocks.ContainerTasks{}

	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil).Once()
	md.On("CreateContainer", mock.Anything).Once().Return("abc", nil)
	md.On("PullImage", mock.Anything, mock.Anything).Once().Return(nil)
	md.On("CreateVolume", "images").Once().Return("images", nil)
	md.On("CopyFileToContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("CopyFilesToVolume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Once().Return(nil, nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything).Return(nil)
	md.On("AttachNetwork", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return cc, md
}

func TestImageCacheCreateDoesNotCreateContainerWhenExists(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Once().Return([]string{"abc"}, nil)

	md.AssertNotCalled(t, "CreateContainer", "images")
}

func TestImageCacheCreateCreatesVolume(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	md.AssertCalled(t, "CreateVolume", "images")
}

func TestImageCachePullsImage(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	md.AssertCalled(t, "PullImage", ctypes.Image{Name: cacheImage}, false)
}

func TestImageCacheCreateAddsVolumes(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*ctypes.Container)

	// check volumes
	require.Equal(t, utils.FQDNVolumeName("images"), conf.Volumes[0].Source)
	require.Equal(t, "/cache", conf.Volumes[0].Destination)
	require.Equal(t, "volume", conf.Volumes[0].Type)
}

func TestImageCacheCreateAddsEnvironmentVariables(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*ctypes.Container)

	// check environment variables
	require.Equal(t, conf.Environment["CA_KEY_FILE"], "/cache/ca/root.key")
	require.Equal(t, conf.Environment["CA_CRT_FILE"], "/cache/ca/root.cert")
	require.Equal(t, conf.Environment["DEBUG"], "false")
	require.Equal(t, conf.Environment["DEBUG_NGINX"], "false")
	require.Equal(t, conf.Environment["DEBUG_HUB"], "false")
	require.Equal(t, conf.Environment["DOCKER_MIRROR_CACHE"], "/cache/docker")
	require.Equal(t, conf.Environment["ENABLE_MANIFEST_CACHE"], "true")
	require.Equal(t, conf.Environment["REGISTRIES"], defaultRegistries)
	require.Equal(t, conf.Environment["AUTH_REGISTRY_DELIMITER"], ":::")
	require.Equal(t, conf.Environment["AUTH_REGISTRIES"], "")
	require.Equal(t, conf.Environment["ALLOW_PUSH"], "true")
	require.Equal(t, conf.Environment["VERIFY_SSL"], "false")
}

func TestImageCacheCreateAddsUnauthenticatedRegistries(t *testing.T) {
	cc, md := setupImageCacheTests(t)
	cc.Registries = []Registry{
		Registry{
			Hostname: "my.registry",
		},
		Registry{
			Hostname: "my.other.registry",
		},
	}

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*ctypes.Container)

	require.Equal(t, conf.Environment["REGISTRIES"], defaultRegistries+" my.registry my.other.registry")
	require.Equal(t, conf.Environment["AUTH_REGISTRIES"], "")
}

func TestImageCacheCreateAddsAuthenticatedRegistries(t *testing.T) {
	cc, md := setupImageCacheTests(t)
	cc.Registries = []Registry{
		Registry{
			Hostname: "my.registry",
			Auth: &RegistryAuth{
				Username: "user1",
				Password: "password1",
			},
		},
		Registry{
			Hostname: "my.other.registry",
			Auth: &RegistryAuth{
				Hostname: "alt.domain.registry",
				Username: "user2",
				Password: "password2",
			},
		},
	}

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*ctypes.Container)

	require.Equal(t, conf.Environment["REGISTRIES"], defaultRegistries+" my.registry my.other.registry")
	require.Equal(t, conf.Environment["AUTH_REGISTRIES"], "my.registry:::user1:::password1 alt.domain.registry:::user2:::password2")
}

func TestImageCacheCreateCopiesCerts(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	// check copies certs
	md.AssertCalled(
		t,
		"CopyFilesToVolume",
		"images",
		[]string{
			filepath.Join(utils.CertsDir(""), "root.cert"),
			filepath.Join(utils.CertsDir(""), "root.key"),
		},
		"/ca",
		true,
	)
}

func TestImageCacheAttachesAndDetatchesNetworks(t *testing.T) {
	cc, md := setupImageCacheTests(t)

	cc.DependsOn = []string{"resource.network.one", "resource.network.two"}

	containerJSON := &dtypes.ContainerJSON{}
	json.Unmarshal([]byte(cacheContainerInfoWithNetworks), containerJSON)

	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Once().Return([]string{"abc"}, nil)
	md.On("ContainerInfo", "abc").Once().Return(*containerJSON, nil)
	md.On("FindContainerIDs", mock.Anything).Once().Return([]string{"abc"}, nil)

	md.On("FindNetwork", "resource.network.one").Once().Return(types.NetworkAttachment{Name: "one"}, nil)
	md.On("FindNetwork", "resource.network.two").Once().Return(types.NetworkAttachment{Name: "two"}, nil)

	c := Provider{cc, md, logger.NewTestLogger(t)}
	err := c.Create()
	require.NoError(t, err)

	// should detatch existing cloud network
	md.AssertNumberOfCalls(t, "DetachNetwork", 1)
	md.AssertCalled(t, "DetachNetwork", "cloud", "abc")

	md.AssertNumberOfCalls(t, "AttachNetwork", 2)
	md.AssertCalled(t, "AttachNetwork", "one", "abc", mock.Anything, mock.Anything)
	md.AssertCalled(t, "AttachNetwork", "two", "abc", mock.Anything, mock.Anything)
}

var cacheContainerInfoWithNetworks = `
{
    "Id": "1d77f21c6a497f9c8c861a26caf6b518b5ef4638335f5a394e7b0e6c9a8e54c2",
    "Created": "2022-02-01T05:36:29.571970814Z",
    "Path": "/entrypoint.sh",
    "Args": [
        "/entrypoint.sh"
    ],
    "State": {
        "Status": "running",
        "Running": true,
        "Paused": false,
        "Restarting": false,
        "OOMKilled": false,
        "Dead": false,
        "Pid": 171587,
        "ExitCode": 0,
        "Error": "",
        "StartedAt": "2022-02-01T05:36:30.339590927Z",
        "FinishedAt": "0001-01-01T00:00:00Z"
    },
    "Image": "docker.io/shipyardrun/docker-registry-proxy:0.6.3",
    "ResolvConfPath": "/run/containers/storage/overlay-containers/1d77f21c6a497f9c8c861a26caf6b518b5ef4638335f5a394e7b0e6c9a8e54c2/userdata/resolv.conf",
    "HostnamePath": "/run/containers/storage/overlay-containers/1d77f21c6a497f9c8c861a26caf6b518b5ef4638335f5a394e7b0e6c9a8e54c2/userdata/hostname",
    "HostsPath": "/run/containers/storage/overlay-containers/1d77f21c6a497f9c8c861a26caf6b518b5ef4638335f5a394e7b0e6c9a8e54c2/userdata/hosts",
    "LogPath": "/var/lib/containers/storage/overlay-containers/1d77f21c6a497f9c8c861a26caf6b518b5ef4638335f5a394e7b0e6c9a8e54c2/userdata/ctr.log",
    "Name": "/docker-cache.image-cache.shipyard.run",
    "RestartCount": 0,
    "Driver": "overlay",
    "Platform": "linux",
    "MountLabel": "",
    "ProcessLabel": "",
    "AppArmorProfile": "containers-default-0.38.16",
    "ExecIDs": [],
    "HostConfig": {
        "Binds": [
            "85d574122fb5aa224b2086e6b72f1a3a60e496855b9281773dbef7f1a69f609a:/ca:rprivate,rw,nodev,exec,nosuid,rbind",
            "77c1a944559390955002af5be4ae7da86dd3b51807a46ab3a64401f830cc3c8e:/docker_mirror_cache:rprivate,rw,nodev,exec,nosuid,rbind",
            "images.volume.shipyard.run:/cache:z,rw,rprivate,nosuid,nodev,rbind"
        ],
        "ContainerIDFile": "",
        "LogConfig": {
            "Type": "json-file",
            "Config": null
        },
        "NetworkMode": "bridge",
        "PortBindings": {
            "3128/tcp": [
                {
                    "HostIp": "0.0.0.0",
                    "HostPort": "32649"
                }
            ]
        },
        "RestartPolicy": {
            "Name": "",
            "MaximumRetryCount": 0
        },
        "AutoRemove": false,
        "VolumeDriver": "",
        "VolumesFrom": null,
        "CapAdd": [],
        "CapDrop": [
            "AUDIT_WRITE",
            "MKNOD",
            "NET_RAW"
        ],
        "CgroupnsMode": "",
        "Dns": [],
        "DnsOptions": [],
        "DnsSearch": [],
        "ExtraHosts": [],
        "GroupAdd": [],
        "IpcMode": "private",
        "Cgroup": "",
        "Links": null,
        "OomScoreAdj": 0,
        "PidMode": "private",
        "Privileged": false,
        "PublishAllPorts": false,
        "ReadonlyRootfs": false,
        "SecurityOpt": [],
        "UTSMode": "private",
        "UsernsMode": "",
        "ShmSize": 65536000,
        "Runtime": "oci",
        "ConsoleSize": [
            0,
            0
        ],
        "Isolation": "",
        "CpuShares": 0,
        "Memory": 0,
        "NanoCpus": 0,
        "CgroupParent": "",
        "BlkioWeight": 0,
        "BlkioWeightDevice": null,
        "BlkioDeviceReadBps": null,
        "BlkioDeviceWriteBps": null,
        "BlkioDeviceReadIOps": null,
        "BlkioDeviceWriteIOps": null,
        "CpuPeriod": 0,
        "CpuQuota": 0,
        "CpuRealtimePeriod": 0,
        "CpuRealtimeRuntime": 0,
        "CpusetCpus": "",
        "CpusetMems": "",
        "Devices": [],
        "DeviceCgroupRules": null,
        "DeviceRequests": null,
        "KernelMemory": 0,
        "KernelMemoryTCP": 0,
        "MemoryReservation": 0,
        "MemorySwap": 0,
        "MemorySwappiness": 0,
        "OomKillDisable": false,
        "PidsLimit": 2048,
        "Ulimits": [
            {
                "Name": "RLIMIT_NOFILE",
                "Hard": 1048576,
                "Soft": 1048576
            },
            {
                "Name": "RLIMIT_NPROC",
                "Hard": 4194304,
                "Soft": 4194304
            }
        ],
        "CpuCount": 0,
        "CpuPercent": 0,
        "IOMaximumIOps": 0,
        "IOMaximumBandwidth": 0,
        "MaskedPaths": null,
        "ReadonlyPaths": null
    },
    "GraphDriver": {
        "Data": {
            "LowerDir": "/var/lib/containers/storage/overlay/0886c5ba112bbf2709e1c7d8b174c683644f7ec135e67b933f627eaad0718a2d/diff:/var/lib/containers/storage/overlay/ee1dd8ef6889154dccb4e49158982000e48cb41bb76753561d1c138bfbfe35f1/diff:/var/lib/containers/storage/overlay/6fe6cce1e3bded1904075577d88c70fe36e0b07f4e4d27936b3479637b4939b0/diff:/var/lib/containers/storage/overlay/37317c100c3b51135c8741a1396e9c7f6c3d0bbce8e925d1b3f6db19f62a018c/diff:/var/lib/containers/storage/overlay/5e06a8652e0820e0ed4569e7e2513bd34d406d781c85f3d96e3610584f4ef4db/diff:/var/lib/containers/storage/overlay/706712cdc2f71496c2c1f89892d454f2d6cf1e0c2895d0771b0727f306c69c35/diff:/var/lib/containers/storage/overlay/aaf1c644ffc7c20f1f15610121d0ce9b8e0b3a58832630b8b730d22dfcb66ee9/diff:/var/lib/containers/storage/overlay/b7a2886645fae735275c2f90db90353c61d0c98f4cb08ce5ef8a618a5fc2563c/diff:/var/lib/containers/storage/overlay/2991f5ba1bd0d984f8df996c308ddd0e4bb3e86c5ed6c316c7ff5cf78a6a3740/diff:/var/lib/containers/storage/overlay/b8410461137f355380c5b708f4f9ca8498b9cfbf2890d99ffe89b30dec7b81d6/diff:/var/lib/containers/storage/overlay/f178fad2d09999ca2c9b85a581c2060f8ac60feafcff38e89bb8d8fbf3a1a51a/diff:/var/lib/containers/storage/overlay/240fbe1c223124aea4bd56f5672c7361f9f5a26f412660e6fd54174ff065f588/diff:/var/lib/containers/storage/overlay/6841f8e260f700a90cb8b81d011c8d19607635fc0fe4d2acaa8fbb7d0dfe1c0d/diff:/var/lib/containers/storage/overlay/13d49216d316caf55f8ff2d074159597d016503477f99791d24f834a0fd92a3c/diff:/var/lib/containers/storage/overlay/cd3761cfa1d9a67e63677d272007823b2b19f90706286017eba48c79a1b3381c/diff",
            "MergedDir": "/var/lib/containers/storage/overlay/8d775934cfffd32689b2878095ff319e531ac920f66c8dc10b82b1c7aa7d5b1d/merged",
            "UpperDir": "/var/lib/containers/storage/overlay/8d775934cfffd32689b2878095ff319e531ac920f66c8dc10b82b1c7aa7d5b1d/diff",
            "WorkDir": "/var/lib/containers/storage/overlay/8d775934cfffd32689b2878095ff319e531ac920f66c8dc10b82b1c7aa7d5b1d/work"
        },
        "Name": "overlay"
    },
    "SizeRootFs": 0,
    "Mounts": [
        {
            "Type": "volume",
            "Name": "85d574122fb5aa224b2086e6b72f1a3a60e496855b9281773dbef7f1a69f609a",
            "Source": "/var/lib/containers/storage/volumes/85d574122fb5aa224b2086e6b72f1a3a60e496855b9281773dbef7f1a69f609a/_data",
            "Destination": "/ca",
            "Driver": "local",
            "Mode": "",
            "RW": true,
            "Propagation": "rprivate"
        },
        {
            "Type": "volume",
            "Name": "77c1a944559390955002af5be4ae7da86dd3b51807a46ab3a64401f830cc3c8e",
            "Source": "/var/lib/containers/storage/volumes/77c1a944559390955002af5be4ae7da86dd3b51807a46ab3a64401f830cc3c8e/_data",
            "Destination": "/docker_mirror_cache",
            "Driver": "local",
            "Mode": "",
            "RW": true,
            "Propagation": "rprivate"
        },
        {
            "Type": "volume",
            "Name": "images.volume.shipyard.run",
            "Source": "/var/lib/containers/storage/volumes/images.volume.shipyard.run/_data",
            "Destination": "/cache",
            "Driver": "local",
            "Mode": "z",
            "RW": true,
            "Propagation": "rprivate"
        }
    ],
    "Config": {
        "Hostname": "docker-cache",
        "Domainname": "",
        "User": "",
        "AttachStdin": false,
        "AttachStdout": false,
        "AttachStderr": false,
        "ExposedPorts": {
            "3128/tcp": {}
        },
        "Tty": false,
        "OpenStdin": false,
        "StdinOnce": false,
        "Env": [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "TERM=xterm",
            "container=podman",
            "ENABLE_MANIFEST_CACHE=true",
            "PROXY_CONNECT_SEND_TIMEOUT=60s",
            "DEBUG_HUB=false",
            "MANIFEST_CACHE_SECONDARY_REGEX=(.*)(\\d|\\.)+(.*)(\\d|\\.)+(.*)(\\d|\\.)+",
            "VERIFY_SSL=true",
            "DEBUG_NGINX=false",
            "MANIFEST_CACHE_PRIMARY_REGEX=(stable|nightly|production|test)",
            "MANIFEST_CACHE_DEFAULT_TIME=1h",
            "MANIFEST_CACHE_SECONDARY_TIME=60d",
            "LANG=en_US.UTF-8",
            "KEEPALIVE_TIMEOUT=300s",
            "ALLOW_PUSH=true",
            "PROXY_SEND_TIMEOUT=60s",
            "DO_DEBUG_BUILD=1",
            "NGINX_VERSION=1.18.0",
            "DOCKER_MIRROR_CACHE=/cache/docker",
            "PROXY_CONNECT_CONNECT_TIMEOUT=60s",
            "DEBUG=false",
            "PROXY_CONNECT_TIMEOUT=60s",
            "CLIENT_HEADER_TIMEOUT=60s",
            "PROXY_READ_TIMEOUT=60s",
            "PROXY_CONNECT_READ_TIMEOUT=60s",
            "MANIFEST_CACHE_PRIMARY_TIME=10m",
            "AUTH_REGISTRIES=some.authenticated.registry:oneuser:onepassword another.registry:user:password",
            "REGISTRIES=k8s.gcr.io gcr.io asia.gcr.io eu.gcr.io us.gcr.io quay.io ghcr.io docker.pkg.github.com",
            "SEND_TIMEOUT=60s",
            "CLIENT_BODY_TIMEOUT=60s",
            "CA_KEY_FILE=/cache/ca/root.key",
            "CA_CRT_FILE=/cache/ca/root.cert",
            "HOSTNAME=docker-cache",
            "HOME=/root"
        ],
        "Cmd": [],
        "Image": "docker.io/shipyardrun/docker-registry-proxy:0.6.3",
        "Volumes": null,
        "WorkingDir": "/",
        "Entrypoint": [
            "/entrypoint.sh"
        ],
        "OnBuild": null,
        "Labels": {
            "org.opencontainers.image.source": "https://github.com/rpardini/docker-registry-proxy"
        },
        "StopSignal": "15",
        "StopTimeout": 0
    },
    "NetworkSettings": {
        "Bridge": "",
        "SandboxID": "",
        "HairpinMode": false,
        "LinkLocalIPv6Address": "",
        "LinkLocalIPv6PrefixLen": 0,
        "Ports": {
            "3128/tcp": [
                {
                    "HostIp": "0.0.0.0",
                    "HostPort": "32649"
                }
            ]
        },
        "SandboxKey": "/run/netns/cni-6547a524-8716-6f36-afdf-50c85a3151cc",
        "SecondaryIPAddresses": null,
        "SecondaryIPv6Addresses": null,
        "EndpointID": "",
        "Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "IPAddress": "",
        "IPPrefixLen": 0,
        "IPv6Gateway": "",
        "MacAddress": "",
        "Networks": {
            "cloud": {
                "IPAMConfig": null,
                "Links": null,
                "Aliases": null,
                "NetworkID": "cloud",
                "EndpointID": "",
                "Gateway": "10.5.0.1",
                "IPAddress": "10.5.0.33",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "MacAddress": "42:bf:b6:3e:80:60",
                "DriverOpts": null
            }
        }
    }
}

`
