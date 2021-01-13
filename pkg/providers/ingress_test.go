package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testIngressCreateMocks() (*mocks.MockContainerTasks, *config.Config) {
	md := &mocks.MockContainerTasks{}
	md.On("PullImage", mock.Anything, mock.Anything).Return(nil)
	md.On("CreateContainer", mock.Anything).Return("ingress", nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("CopyFileToContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	testCluster.Driver = "k3s"

	c := config.New()

	tic := testIngressConfig
	c.AddResource(&tic)

	tkc := testK8sIngressConfig
	c.AddResource(&tkc)

	tnc := testNomadIngressConfig
	c.AddResource(&tnc)

	tcc := testIngressContainerConfig
	c.AddResource(&tcc)

	tc := *testCluster
	tc.Driver = "k3s"
	c.AddResource(&tc)

	tn := *testNomadCluster
	c.AddResource(&tn)

	tco := *testContainer
	c.AddResource(&tco)

	return md, c
}

func TestIngressK8sErrorsWhenUnableToLookupIDs(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	p := NewK8sIngress(&testK8sIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressK8sErrorsWhenClusterExists(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)

	p := NewK8sIngress(&testK8sIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressK8sTargetPullsImage(t *testing.T) {
	md, c := testIngressCreateMocks()
	conf, _ := c.FindResource("k8s_ingress.web-http")

	p := NewK8sIngress(conf.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: ingressImage}, false)
}

func TestIngressK8sTargetConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	conf, _ := c.FindResource("k8s_ingress.web-http")

	p := NewK8sIngress(conf.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)
	// ensure the proxy is type k8s
	assert.Equal(t, "--proxy-type", params.Command[0])
	assert.Equal(t, "kubernetes", params.Command[1])

	// default namespace should be default
	assert.Equal(t, "default", params.Command[3])

	// check the servicename
	assert.Equal(t, "--service-name", params.Command[4])
	assert.Equal(t, testIngressConfig.Service, params.Command[5])
}

func TestIngressK8sTargetConfiguresKubeConfig(t *testing.T) {
	md, c := testIngressCreateMocks()
	conf, _ := c.FindResource("k8s_ingress.web-http")

	p := NewK8sIngress(conf.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	container := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check the env var for the kubeconfig is set
	assert.Equal(t, "/kubeconfig-docker.yaml", container.Environment[0].Value)

	// check that the kubeconfig has been copied to the container
	params := getCalls(&md.Mock, "CopyFileToContainer")[0].Arguments
	assert.Equal(t, "ingress", params[0])
	assert.Contains(t, params[1], ".shipyard/config/test/kubeconfig-docker.yaml")
	assert.Equal(t, "/", params[2])
}

func TestIngressK8sTargetWithNamespaceConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("k8s_ingress.web-http")

	tc.(*config.K8sIngress).Namespace = "mine"
	p := NewK8sIngress(tc.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "mine", params.Command[3])
}

func TestIngressK8sTargetWithServiceConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("k8s_ingress.web-http")

	tc.(*config.K8sIngress).Service = "myservice"
	p := NewK8sIngress(tc.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "svc/myservice", params.Command[5])
}

func TestIngressK8sTargetWithPodConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("k8s_ingress.web-http")

	tc.(*config.K8sIngress).Service = ""
	tc.(*config.K8sIngress).Pod = "mypod"
	p := NewK8sIngress(tc.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "mypod", params.Command[5])
}

func TestIngressK8sTargetWithDeploymentConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("k8s_ingress.web-http")

	tc.(*config.K8sIngress).Deployment = "mydeployment"
	tc.(*config.K8sIngress).Service = ""
	p := NewK8sIngress(tc.(*config.K8sIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "deployment/mydeployment", params.Command[5])
}

func TestIngressContainerTargetConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("container_ingress.web-http")

	p := NewContainerIngress(tc.(*config.ContainerIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	assert.Equal(t, "--service-name", params.Command[0])
	assert.Equal(t, "test.container.shipyard.run", params.Command[1])
}

func TestIngressContainerAddsPorts(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("container_ingress.web-http")

	p := NewContainerIngress(tc.(*config.ContainerIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)
	assert.Equal(t, "--ports", params.Command[2])
	assert.Equal(t, "8080:8081", params.Command[3])

	assert.Equal(t, "--ports", params.Command[4])
	assert.Equal(t, "9080:9081", params.Command[5])

	// check the host ports
	assert.Equal(t, testIngressContainerConfig.Ports, params.Ports)
}

func TestIngressContainerFailReturnsError(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("container_ingress.web-http")

	removeOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Return("", fmt.Errorf("boom"))
	p := NewContainerIngress(tc.(*config.ContainerIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressK8sTargetDestroysContainer(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("ingress.web-http")

	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"ingress"}, nil)
	p := NewIngress(tc.(*config.Ingress), md, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", "ingress")
	md.AssertCalled(t, "DetachNetwork", mock.Anything, mock.Anything, mock.Anything)
}

func TestIngressNomadTargetConfiguresNomadConfig(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("nomad_ingress.web-http")

	p := NewNomadIngress(tc.(*config.NomadIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check the volume mount is set
	_, path := utils.CreateClusterConfigPath("test")
	assert.Equal(t, path, params.Volumes[0].Source)
	assert.Equal(t, "/.nomad/config.json", params.Volumes[0].Destination)
}

func TestIngressNomadTargetConfiguresCommand(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("nomad_ingress.web-http")

	p := NewNomadIngress(tc.(*config.NomadIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)
	// ensure the proxy is type k8s
	assert.Equal(t, "--proxy-type", params.Command[0])
	assert.Equal(t, "nomad", params.Command[1])

	assert.Equal(t, "--nomad-config", params.Command[2])
	assert.Equal(t, "/.nomad/config.json", params.Command[3])

	// check the servicename
	assert.Equal(t, "--service-name", params.Command[4])
	assert.Equal(t, "test.group.task", params.Command[5])
}

func TestIngressNomadAddsPorts(t *testing.T) {
	md, c := testIngressCreateMocks()
	tc, _ := c.FindResource("nomad_ingress.web-http")

	p := NewNomadIngress(tc.(*config.NomadIngress), md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)
	assert.Equal(t, "--ports", params.Command[6])
	assert.Equal(t, "8080:http", params.Command[7])

	// check the host ports
	assert.Equal(t, tc.(*config.NomadIngress).Ports, params.Ports)
}

var testIngressConfig = config.Ingress{
	Service: "svc/web",
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
		Type: config.TypeIngress,
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Target:   "k8s_cluster.test",
}

var testK8sIngressConfig = config.K8sIngress{
	Service: "web",
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
		Type: config.TypeK8sIngress,
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Cluster:  "k8s_cluster.test",
}

var testNomadIngressConfig = config.NomadIngress{
	Job:   "test",
	Group: "group",
	Task:  "task",
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
		Type: config.TypeNomadIngress,
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Cluster:  "nomad_cluster.test",
	Ports: []config.Port{
		config.Port{
			Local:  "8080",
			Remote: "http",
			Host:   "8082",
		},
	},
}

var testIngressContainerConfig = config.ContainerIngress{
	Target: "container.test",
	Ports: []config.Port{
		config.Port{
			Local:  "8080",
			Remote: "8081",
			Host:   "8082",
		},
		config.Port{
			Local:  "9080",
			Remote: "9081",
			Host:   "9082",
		},
	},
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
		Type: config.TypeContainerIngress,
	},
}

var testContainer = config.NewContainer("test")
var testCluster = config.NewK8sCluster("test")
var testNomadCluster = config.NewNomadCluster("test")
