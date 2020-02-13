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
	md.On("CreateContainer", mock.Anything).Return("ingress", nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"ingress"}, nil)
	return md
}

func TestIngressK8sTargetConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewIngress(testIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)
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
	p := NewIngress(testIngressConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)

	// check the volume mount is set
	_, _, path := utils.CreateKubeConfigPath("test")
	assert.Equal(t, path, params.Volumes[0].Source)
	assert.Equal(t, "/.kube/kubeconfig.yml", params.Volumes[0].Destination)

	// check the env var for the kubeconfig is set
	assert.Equal(t, "/.kube/kubeconfig.yml", params.Environment[0].Value)
}

func TestIngressK8sTargetWithNamespaceConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	tc := testIngressConfig
	tc.Namespace = "mine"
	p := NewIngress(tc, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)

	// namespace should be same as custom
	assert.Equal(t, "mine", params.Command[3])
}

func TestIngressContainerTargetConfiguresCommand(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewIngress(testIngressContainerConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)

	assert.Equal(t, "--service-name", params.Command[0])
	assert.Equal(t, "test.cloud.shipyard", params.Command[1])
}

func TestIngressContainerAddsPorts(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewIngress(testIngressContainerConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(config.Container)
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
	p := NewIngress(testIngressContainerConfig, md, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressK8sTargetDestroysContainer(t *testing.T) {
	md := testIngressCreateMocks()
	p := NewIngress(testIngressConfig, md, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)
	md.AssertCalled(t, "RemoveContainer", "ingress")
}

var testIngressConfig = config.Ingress{
	Service:    "svc/web",
	ResourceInfo: config.ResourceInfo{
		Name:       "web-http",
	},
	Networks: []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}},
	Target: "k8s_cluster.test",
}

var testIngressContainerConfig = config.Ingress{
	Target: "container.test",
	Ports: []config.Port{
		config.Port{
			Local:  8080,
			Remote: 8081,
			Host:   8082,
		},
		config.Port{
			Local:  9080,
			Remote: 9081,
			Host:   9082,
		},
	},
}
