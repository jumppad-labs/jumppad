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

func testIngressCreateMocks() *mocks.MockContainerTasks {
	md := &mocks.MockContainerTasks{}
	md.On("PullImage", mock.Anything, mock.Anything).Return(nil)
	md.On("CreateContainer", mock.Anything).Return("ingress", nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{}, nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	testCluster.Driver = "k3s"

	c := config.New()
	c.AddResource(&testK8sIngressConfig)
	c.AddResource(&testIngressConfig)
	c.AddResource(&testIngressContainerConfig)
	c.AddResource(testCluster)
	c.AddResource(testContainer)

	return md
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
	md := testIngressCreateMocks()
	p := NewK8sIngress(&testK8sIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)
	md.AssertCalled(t, "PullImage", config.Image{Name: ingressImage}, false)
}

func TestIngressK8sTargetConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewK8sIngress(&testK8sIngressConfig, md, hclog.NewNullLogger())

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
	md := testIngressCreateMocks()
	p := NewK8sIngress(&testK8sIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check the volume mount is set
	_, _, path := utils.CreateKubeConfigPath("test")
	assert.Equal(t, path, params.Volumes[0].Source)
	assert.Equal(t, "/.kube/kubeconfig.yml", params.Volumes[0].Destination)

	// check the env var for the kubeconfig is set
	assert.Equal(t, "/.kube/kubeconfig.yml", params.Environment[0].Value)
}

func TestIngressK8sTargetWithNamespaceConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	tc := testK8sIngressConfig
	tc.Namespace = "mine"
	p := NewK8sIngress(&tc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "mine", params.Command[3])
}

func TestIngressK8sTargetWithServiceConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	tc := testK8sIngressConfig
	tc.Service = "myservice"
	p := NewK8sIngress(&tc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "svc/myservice", params.Command[5])
}

func TestIngressK8sTargetWithPodConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	tc := testK8sIngressConfig
	tc.Service = ""
	tc.Pod = "mypod"
	p := NewK8sIngress(&tc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "mypod", params.Command[5])
}

func TestIngressK8sTargetWithDeploymentConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	tc := testK8sIngressConfig
	tc.Service = ""
	tc.Deployment = "mydeployment"
	p := NewK8sIngress(&tc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// namespace should be same as custom
	assert.Equal(t, "deployment/mydeployment", params.Command[5])
}

func TestIngressContainerTargetConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewContainerIngress(&testIngressContainerConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	assert.Equal(t, "--service-name", params.Command[0])
	assert.Equal(t, "test.container.shipyard.run", params.Command[1])
}

func TestIngressContainerAddsPorts(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewContainerIngress(&testIngressContainerConfig, md, hclog.NewNullLogger())

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
	md := testIngressCreateMocks()
	removeOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Return("", fmt.Errorf("boom"))
	p := NewContainerIngress(&testIngressContainerConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressK8sTargetDestroysContainer(t *testing.T) {
	md := testIngressCreateMocks()
	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"ingress"}, nil)
	p := NewIngress(&testIngressConfig, md, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", "ingress")
	md.AssertCalled(t, "DetachNetwork", mock.Anything, mock.Anything, mock.Anything)
}

var testIngressConfig = config.Ingress{
	Service: "svc/web",
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Target:   "k8s_cluster.test",
}

var testK8sIngressConfig = config.K8sIngress{
	Service: "web",
	ResourceInfo: config.ResourceInfo{
		Name: "web-http",
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Cluster:  "k8s_cluster.test",
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
}

var testContainer = config.NewContainer("test")
var testCluster = config.NewK8sCluster("test")
