package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	conmocks "github.com/jumppad-labs/jumppad/pkg/clients/connector/mocks"
	contypes "github.com/jumppad-labs/jumppad/pkg/clients/connector/types"
	cmocks "github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/k8s"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"

	container "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/mohae/deepcopy"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

// setupClusterMocks sets up a happy path for mocks
func setupClusterMocks(t *testing.T) (
	*K8sCluster, *cmocks.ContainerTasks, *k8s.MockKubernetes, *conmocks.Connector) {

	md := &cmocks.ContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil).Once()
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"123"}, nil) // second call should find the cluster

	md.On("PullImage", mock.Anything, mock.Anything).Return(nil)
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("123", nil)
	md.On("CreateContainer", mock.Anything).Return("containerid", nil)
	md.On("ContainerLogs", mock.Anything, true, true).Return(
		io.NopCloser(bytes.NewBufferString("Running kubelet")),
		nil,
	)
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything).Return([]string{"/images/file.tar.gz"}, nil)
	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveContainer", mock.Anything, mock.Anything).Return(nil)
	md.On("RemoveVolume", mock.Anything).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("ListNetworks", mock.Anything).Return([]types.NetworkAttachment{})

	md.On("EngineInfo").Return(&types.EngineInfo{StorageDriver: "overlay2"})

	// set the home folder to a temp folder
	tmpDir := t.TempDir()
	currentHome := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), tmpDir)

	// create the fake cert
	certfile := filepath.Join(utils.CertsDir(""), "/root.cert")
	cf, err := os.Create(certfile)
	if err != nil {
		panic(err)
	}
	cf.WriteString("CA")
	cf.Close()

	// write the kubeconfig
	_, kubePath, _ := utils.CreateKubeConfigPath(clusterConfig.Meta.ID)
	kcf, err := os.Create(kubePath)
	if err != nil {
		panic(err)
	}
	kcf.WriteString(kubeconfig)
	kcf.Close()

	// create the Kubernetes client mock
	mk := &k8s.MockKubernetes{}
	mk.Mock.On("SetConfig", mock.Anything).Return(nil)
	mk.Mock.On("HealthCheckPods", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.Mock.On("Apply", mock.Anything, mock.Anything).Return(nil)
	mk.Mock.On("GetPodLogs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	mc := &conmocks.Connector{}
	mc.On("GetLocalCertBundle", mock.Anything).Return(&contypes.CertBundle{}, nil)
	mc.On("GenerateLeafCert",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(&contypes.CertBundle{}, nil)

	// copy the config
	cc := deepcopy.Copy(clusterConfig).(*K8sCluster)

	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), currentHome)
	})

	return cc, md, mk, mc
}

func TestClusterK3ErrorsWhenUnableToLookupIDs(t *testing.T) {
	md := &cmocks.ContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	mk := &k8s.MockKubernetes{}
	p := ClusterProvider{clusterConfig, md, mk, nil, nil, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3SetsEnvironment(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	cc.Image = nil

	p := ClusterProvider{clusterConfig, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*types.Container)

	assert.Equal(t, params.Environment["K3S_KUBECONFIG_OUTPUT"], "/output/kubeconfig.yaml")
	assert.Equal(t, params.Environment["CONTAINERD_HTTP_PROXY"], utils.ImageCacheAddress())
	assert.Equal(t, params.Environment["CONTAINERD_HTTPS_PROXY"], utils.ImageCacheAddress())
	assert.Equal(t, params.Environment["CONTAINERD_NO_PROXY"], "")
	assert.Equal(t, params.Environment["PROXY_CA"], "CA")
}

func TestClusterK3DoesNotSetProxyEnvironmentWithWrongVersion(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	cc.Image = &container.Image{Name: "jumppad.dev/k3s:v1.12.1"}

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*types.Container)

	assert.Empty(t, params.Environment["CONTAINERD_HTTP_PROXY"])
}

func TestClusterK3NoProxyIsSet(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	cc.Config = &Config{DockerConfig: &DockerConfig{NoProxy: []string{"test.com", "test2.com"}}}

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*types.Container)
	assert.Equal(t, "test.com,test2.com", params.Environment["CONTAINERD_NO_PROXY"])
}

func TestClusterK3ErrorsWhenClusterExists(t *testing.T) {
	md := &cmocks.ContainerTasks{}
	md.On("FindContainerIDs", utils.FQDN("server."+clusterConfig.Meta.Name, "", TypeK8sCluster)).Return([]string{"abc"}, nil)

	mk := &k8s.MockKubernetes{}
	p := ClusterProvider{clusterConfig, md, mk, nil, nil, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3PullsImage(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", types.Image{Name: "shipyardrun/k3s:v1.27.4"}, false)
}

func TestClusterK3CreatesNewVolume(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	md.AssertCalled(t, "CreateVolume", utils.ImageVolumeName)
}

func TestClusterK3FailsWhenUnableToCreatesANewVolume(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&md.Mock, "CreateVolume")
	md.On("CreateVolume", mock.Anything, mock.Anything).Return("", fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
	md.AssertCalled(t, "CreateVolume", utils.ImageVolumeName)
}

func TestClusterK3CreatesAServer(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*types.Container)

	// validate the basic details for the server container
	assert.Contains(t, params.Name, "server")
	assert.Contains(t, params.Image.Name, "shipyardrun")
	assert.Equal(t, "cloud", params.Networks[0].ID)
	assert.True(t, params.Privileged)

	// validate that the volume is correctly set
	assert.Equal(t, "123", params.Volumes[0].Source)
	assert.Equal(t, "/cache", params.Volumes[0].Destination)
	assert.Equal(t, "volume", params.Volumes[0].Type)

	// validate the API port is set
	localPort, _ := strconv.Atoi(params.Ports[0].Local)
	hostPort, _ := strconv.Atoi(params.Ports[0].Host)
	assert.Equal(t, localPort, 443)
	assert.Equal(t, hostPort, localPort)
	assert.Equal(t, "tcp", params.Ports[0].Protocol)

	localPort, _ = strconv.Atoi(params.Ports[1].Local)
	hostPort, _ = strconv.Atoi(params.Ports[1].Host)
	assert.GreaterOrEqual(t, hostPort, utils.MinRandomPort)
	assert.LessOrEqual(t, hostPort, utils.MaxRandomPort)
	assert.Equal(t, localPort, hostPort)
	assert.Equal(t, "tcp", params.Ports[1].Protocol)

	localPort2, _ := strconv.Atoi(params.Ports[2].Local)
	hostPort2, _ := strconv.Atoi(params.Ports[2].Host)
	assert.Equal(t, localPort2, localPort+1)
	assert.Equal(t, localPort2, hostPort2)
	assert.GreaterOrEqual(t, hostPort2, 30000)
	assert.Equal(t, "tcp", params.Ports[2].Protocol)

	// validate the command
	assert.Equal(t, "server", params.Command[0])
	assert.Contains(t, params.Command[1], params.Ports[0].Local)
	assert.Contains(t, params.Command[2], "--kube-proxy-arg=conntrack-max-per-core=0")
	assert.Contains(t, params.Command[3], "--disable=traefik")
	assert.Contains(t, params.Command[4], "--snapshotter=overlayfs")
	assert.Contains(t, params.Command[5], "--tls-san=server.test.k8s-cluster.local.jmpd.in")
}

func TestClusterK3CreatesAServerWithAdditionalPorts(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	cc.Ports = []container.Port{{Local: "8080", Remote: "8080", Host: "8080"}}
	cc.PortRanges = []container.PortRange{{Range: "8000-9000", EnableHost: true}}

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*types.Container)

	localPort, _ := strconv.Atoi(params.Ports[3].Local)
	hostPort, _ := strconv.Atoi(params.Ports[3].Host)
	assert.Equal(t, localPort, 8080)
	assert.Equal(t, hostPort, 8080)

	assert.Equal(t, params.PortRanges[0].Range, "8000-9000")
	assert.True(t, params.PortRanges[0].EnableHost)
}

func TestClusterK3sErrorsIfServerNOTStart(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&md.Mock, "ContainerLogs")
	md.On("ContainerLogs", mock.Anything, true, true).Return(
		ioutil.NopCloser(bytes.NewBufferString("Not running")),
		nil,
	)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}
	startTimeout = 10 * time.Millisecond // reset the startTimeout, do not want to wait 120s

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sDownloadsConfig(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	_, kubePath, _ := utils.CreateKubeConfigPath(cc.Meta.ID)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	params := testutils.GetCalls(&md.Mock, "CopyFromContainer")[0].Arguments
	assert.Equal(t, "containerid", params.String(0))
	assert.Equal(t, "/output/kubeconfig.yaml", params.String(1))
	assert.Equal(t, kubePath, params.String(2))
}

func TestClusterK3sRaisesErrorWhenUnableToDownloadConfig(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&md.Mock, "CopyFromContainer")
	md.On("CopyFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sSetsServerInConfig(t *testing.T) {
	dh := os.Getenv("DOCKER_HOST")
	os.Setenv("DOCKER_HOST", "tcp://localhost")

	t.Cleanup(func() {
		os.Setenv("DOCKER_HOST", dh)
	})

	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	// check the kubeconfig file for docker uses a network ip not localhost

	// check file has been written
	_, kubePath, _ := utils.CreateKubeConfigPath(clusterConfig.Meta.ID)
	f, err := os.Open(kubePath)
	assert.NoError(t, err)
	defer f.Close()

	// check file contains docker ip
	d, err := ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.Contains(t, string(d), "https://"+utils.GetDockerIP())
}

func TestCreateSetsKubeConfig(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	assert.NotEmpty(t, cc.KubeConfig.ConfigPath)
	assert.NotEmpty(t, cc.KubeConfig.CA)
	assert.NotEmpty(t, cc.KubeConfig.ClientCertificate)
	assert.NotEmpty(t, cc.KubeConfig.ClientKey)
}

func TestClusterK3sCreatesKubeClient(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	mk.AssertCalled(t, "SetConfig", mock.Anything)
}

func TestClusterK3sErrorsWhenFailedToCreateKubeClient(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&mk.Mock, "SetConfig")
	mk.Mock.On("SetConfig", mock.Anything).Return(fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sWaitsForPods(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	mk.AssertCalled(t, "HealthCheckPods", mock.Anything, []string{"app=local-path-provisioner", "k8s-app=kube-dns"}, startTimeout)
}

func TestClusterK3sErrorsWhenWaitsForPodsFail(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&mk.Mock, "HealthCheckPods")
	mk.On("HealthCheckPods", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sStreamsLogsWhenRunning(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	mk.On("GetPodLogs", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	logReader, err := mk.GetPodLogs(context.TODO(), mock.Anything, mock.Anything)
	assert.NoError(t, err)
	assert.NotEmpty(t, logReader)
	assert.NoError(t, logReader.Close())
}

func TestClusterK3sImportDockerImagesDoesNothingWhenEmpty(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	cc.CopyImages = append(cc.CopyImages, container.Image{Name: ""})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	md.AssertNumberOfCalls(t, "PullImage", 2)
	md.AssertNumberOfCalls(t, "ExecuteCommand", 2) // once for the import, once to prune any build images

	// should not pull for empty image
	md.AssertNotCalled(t, "PullImage", ctypes.Image{Name: cc.CopyImages[0].Name}, false)

	// should pull for non-empty image
	md.AssertCalled(t, "PullImage", ctypes.Image{Name: cc.CopyImages[1].Name}, false)

	// should update the image id from the registry on the struct
	// this enables us to track when the copy image changes so
	// we can copy a new version to the cluster
	assert.Equal(t, "abc123", cc.CopyImages[1].ID)
}

func TestClusterK3sImportDockerImagesPullsImages(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:abc"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	md.AssertNumberOfCalls(t, "PullImage", 3) //once for main image, once for each copy image
	md.AssertCalled(t, "PullImage", ctypes.Image{Name: cc.CopyImages[0].Name}, false)
	md.AssertCalled(t, "PullImage", ctypes.Image{Name: cc.CopyImages[1].Name}, false)
}

func TestClusterK3sImportDockerCopiesImages(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:abc"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)
	md.AssertCalled(t, "CopyLocalDockerImagesToVolume", []string{"test:123", "test:abc"}, utils.FQDNVolumeName(utils.ImageVolumeName), false)
}

func TestClusterK3sImportDockerCopyImageFailReturnsError(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:abc"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	testutils.RemoveOn(&md.Mock, "CopyLocalDockerImagesToVolume")
	md.On("CopyLocalDockerImagesToVolume", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sImportDockerRunsExecCommand(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:abc"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}
	err := p.Create(context.Background())

	assert.NoError(t, err)
	md.AssertCalled(t, "ExecuteCommand", "123", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestClusterK3sImportDockerExecFailReturnsError(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:123"})
	cc.CopyImages = append(cc.CopyImages, container.Image{Name: "test:abc"})

	md.On("ExecuteCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(1, fmt.Errorf("boom"))
	md.On("FindImageInLocalRegistry", mock.Anything).Return("abc123", nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestClusterK3sGeneratesCertsForConnector(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	mc.AssertCalled(t, "GetLocalCertBundle", mock.Anything)
	mc.AssertCalled(t,
		"GenerateLeafCert",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestClusterK3sGeneratesCertsForDeployment(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	//args := testutils.GetCalls(&mk.Mock, "Apply")[0]
	//path := strings.Split(args.Arguments[0].([]string)[0], "/")

	mc.AssertCalled(t, "GenerateLeafCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestClusterK3sDeploysConnector(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	files := []string{
		"namespace.yaml",
		"secret.yaml",
		"rbac.yaml",
		"deployment.yaml",
	}

	mk.AssertCalled(t, "Apply", mock.Anything, true)

	args := testutils.GetCalls(&mk.Mock, "Apply")[0]

	for _, f := range args.Arguments[0].([]string) {
		_, fn := filepath.Split(f)
		assert.Contains(t, files, fn)
	}
}

func TestClusterK3sWaitsForConnectorStart(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	mk.AssertCalled(t, "HealthCheckPods", mock.Anything, []string{"app=connector"}, 60*time.Second)
}

// Destroy Tests
func TestClusterK3sDestroyGetsIDr(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
	md.AssertCalled(t, "FindContainerIDs", "server.test.k8s-cluster.local.jmpd.in")
}

func TestClusterK3sDestroyWithFindIDErrorReturnsError(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Destroy(context.Background(), false)
	assert.Error(t, err)
}

func TestClusterK3sDestroyWithNoIDReturns(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer", mock.Anything, mock.Anything)
}

func TestClusterK3sDestroyRemovesContainer(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", mock.Anything, false)
}

func TestClusterK3sDestroyRemovesConfig(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)
	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)

	dir, _, _ := utils.CreateKubeConfigPath(cc.Meta.Name)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", mock.Anything, false)

	assert.NoDirExists(t, dir)
}

func TestLookupReturnsIDs(t *testing.T) {
	cc, md, mk, mc := setupClusterMocks(t)

	testutils.RemoveOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"found"}, nil)

	p := ClusterProvider{cc, md, mk, nil, mc, logger.NewTestLogger(t)}

	ids, err := p.Lookup()

	assert.NoError(t, err)
	assert.Equal(t, []string{"found"}, ids)
}

var clusterConfig = &K8sCluster{
	ResourceBase: htypes.ResourceBase{Meta: htypes.Meta{Name: "test", Type: TypeK8sCluster}},
	Image:        &container.Image{Name: "shipyardrun/k3s:v1.27.4"},
	Networks:     []container.NetworkAttachment{container.NetworkAttachment{ID: "cloud"}},
	APIPort:      443,
}

var kubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJWakNCL3FBREFnRUNBZ0VBTUFvR0NDcUdTTTQ5QkFNQ01DTXhJVEFmQmdOVkJBTU1HR3N6Y3kxelpYSjIKWlhJdFkyRkFNVFUzTlRrNE1qVTNNakFlRncweE9URXlNVEF4TWpVMk1USmFGdzB5T1RFeU1EY3hNalUyTVRKYQpNQ014SVRBZkJnTlZCQU1NR0dzemN5MXpaWEoyWlhJdFkyRkFNVFUzTlRrNE1qVTNNakJaTUJNR0J5cUdTTTQ5CkFnRUdDQ3FHU000OUF3RUhBMElBQkhSblYydVliRU53eTlROGkxd2J6ZjQ2NytGdzV2LzRBWVQ2amM4dXorM00KTmRrZEwwd0RhNGM3Y1ByOUFXM1N0ZVRYSDNtNE9mRStJYTE3L1liaDFqR2pJekFoTUE0R0ExVWREd0VCL3dRRQpBd0lDcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUFvR0NDcUdTTTQ5QkFNQ0EwY0FNRVFDSUhFYlZwbUkzbjZwCnQrYlhKaWlFK1hiRm5XUFhtYm40OFZuNmtkYkdPM3daQWlCRDNyUjF5RjQ5R0piZmVQeXBsREdCK3lkNVNQOEUKUmQ4OGxRWW9oRnV2enc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://127.0.0.1:64674
users:
- name: default
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrakNDQVRlZ0F3SUJBZ0lJWmx1UHFJSDFhOHN3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOekE1TmpZek1EUTBNQjRYRFRJME1ETXdOVEU0TWpRd05Gb1hEVEkxTURNdwpOVEU0TWpRd05Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJFemtVS0MweE9XSG1VWkgKdlpSZE5UZ3VLQWZTTlI0VFEwK0NzWXEzNEp4UkhveDIxVlVMaU1OdUp0WWV4QytMTVhBTkt0Zms0Q3N5UytkZwpwWFBKODRpalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCU0FWZkZpcnpKNmNycGpaY2NHNFJ4d3EwOE1uekFLQmdncWhrak9QUVFEQWdOSkFEQkcKQWlFQXhlQmhTZkdIZnlWYjJ4cWNXQzl0b3JkTkpHUGluQ2dMekluK2ZZUEtFOXdDSVFDd3Z6Y1gxVXR0eEQzMwpJZVBVZWxKeHV5ckk2MlNiRUdTNDkwWDRiUUpPd1E9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlCZURDQ0FSMmdBd0lCQWdJQkFEQUtCZ2dxaGtqT1BRUURBakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwClpXNTBMV05oUURFM01EazJOak13TkRRd0hoY05NalF3TXpBMU1UZ3lOREEwV2hjTk16UXdNekF6TVRneU5EQTAKV2pBak1TRXdId1lEVlFRRERCaHJNM010WTJ4cFpXNTBMV05oUURFM01EazJOak13TkRRd1dUQVRCZ2NxaGtqTwpQUUlCQmdncWhrak9QUU1CQndOQ0FBU29RNEdqYjZISC9vQ05zSWgxeEl4dDIwejlKc2dSbjV3aG9mdzl2VlJaCnZZWm9SMk1MOWgyUkVZeDVpNjg0YzdjNGFZOUhRRGNLelZqNVFzMFBpbDBjbzBJd1FEQU9CZ05WSFE4QkFmOEUKQkFNQ0FxUXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QWRCZ05WSFE0RUZnUVVnRlh4WXE4eWVuSzZZMlhIQnVFYwpjS3RQREo4d0NnWUlLb1pJemowRUF3SURTUUF3UmdJaEFKOE5XcjYrWDd2bUVmbStpbHVmdGoxTzVjYlMycmpGClREYkROVGZYQ3JVUkFpRUFnWVBWa2dkS0NEVXRNRWh4N040OFk0UGtkMFhkV0RrUTZaUzlOTERER29zPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUVpQUpQclpTRGVuM0pvaHdyQ2syOXVGbk1FeG10NlRYNU9TckpvS0FaVmVvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFVE9SUW9MVEU1WWVaUmtlOWxGMDFPQzRvQjlJMUhoTkRUNEt4aXJmZ25GRWVqSGJWVlF1SQp3MjRtMWg3RUw0c3hjQTBxMStUZ0t6Skw1MkNsYzhuemlBPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=`
