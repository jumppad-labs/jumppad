package providers

import (
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func testIngressCreateMockConnector(t *testing.T, name string) *clients.ConnectorMock {
	h := os.Getenv(utils.HomeEnvName())
	td := t.TempDir()

	os.Setenv(utils.HomeEnvName(), td)

	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), h)
	})

	utils.GetClusterConfig(string(config.TypeK8sCluster) + ".test")

	m := &clients.ConnectorMock{}
	m.On("ExposeService", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("12345", nil)
	m.On("RemoveService", mock.Anything).Return(nil)

	return m
}

func TestIngressExposeLocalErrorsWhenUnableToFindDependencies(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	tc.Source.Config.Cluster = "blah"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeLocalErrorsWhenInvalidName(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	tc.Name = "connector"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeLocalErrorsWhenInvalidPort(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	tc.Source.Config.Port = "abc"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeLocalErrorsWhenPortReserved(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	tc.Source.Config.Port = "60000"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)

	tc.Source.Config.Port = "60001"

	p = NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err = p.Create()
	assert.Error(t, err)
}

func TestIngressExposeLocalErrorsWhenInvalidAddress(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	tc.Destination.Config.Address = ""
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeLocalCallsExpose(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposeK8sLocalConfig
	c.AddResource(&tc)

	clusterConfig, _ := utils.GetClusterConfig(testIngressExposeK8sLocalConfig.Source.Config.Cluster)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	port, _ := strconv.Atoi(tc.Source.Config.Port)

	mc.AssertCalled(t, "ExposeService",
		tc.Name,
		port,
		clusterConfig.ConnectorAddress(utils.LocalContext),
		tc.Destination.Config.Address+":"+tc.Destination.Config.Port,
		"local")

	assert.Equal(t, tc.Id, "12345")
}

func TestIngressExposeRemoteErrorsWhenUnableToFindDependencies(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, "")

	tc := testIngressExposesLocalK8sServiceConfig
	tc.Destination.Config.Cluster = "blah"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeRemoteErrorsWhenNoDestinationAddress(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, "")

	tc := testIngressExposesLocalK8sServiceConfig
	tc.Destination.Config.Address = ""
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeRemoteErrorsWhenUnableToParsePort(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, "")

	tc := testIngressExposesLocalK8sServiceConfig
	tc.Source.Config.Port = "sdf"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)
}

func TestIngressExposeRemoteErrorsWhenReservedPort(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, "")

	tc := testIngressExposesLocalK8sServiceConfig
	tc.Source.Config.Port = "30001"
	c.AddResource(&tc)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.Error(t, err)

	tc.Source.Config.Port = "30002"

	p = NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err = p.Create()
	assert.Error(t, err)
}

func TestIngressExposeRemoteCallsExpose(t *testing.T) {
	md, c := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposesLocalK8sServiceConfig
	c.AddResource(&tc)

	clusterConfig, _ := utils.GetClusterConfig(testIngressExposeK8sLocalConfig.Source.Config.Cluster)

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Create()
	assert.NoError(t, err)

	port, _ := strconv.Atoi(tc.Source.Config.Port)

	mc.AssertCalled(t, "ExposeService",
		tc.Name,
		port,
		clusterConfig.ConnectorAddress(utils.LocalContext),
		tc.Destination.Config.Address+":"+tc.Destination.Config.Port,
		"remote")

	assert.Equal(t, tc.Id, "12345")
}

func TestIngressDestroyCallsRemove(t *testing.T) {
	md, _ := testIngressCreateMocks()
	mc := testIngressCreateMockConnector(t, testIngressExposeK8sLocalConfig.Name)

	tc := testIngressExposesLocalK8sServiceConfig
	tc.Id = "12345"

	p := NewIngress(&tc, md, mc, hclog.NewNullLogger())

	err := p.Destroy()
	assert.NoError(t, err)

	mc.AssertCalled(t, "RemoveService", "12345")
}

var testIngressExposeK8sLocalConfig = config.Ingress{
	ResourceInfo: config.ResourceInfo{
		Name: "local-http",
		Type: config.TypeIngress,
	},
	Source: config.Traffic{
		Driver: "k8s",

		Config: config.TrafficConfig{
			Cluster: "k8s_cluster.test",
			Port:    "12344",
		},
	},

	Destination: config.Traffic{
		Driver: "local",

		Config: config.TrafficConfig{
			Port:    "1234",
			Address: "localhost",
		},
	},
}

var testIngressExposesLocalK8sServiceConfig = config.Ingress{
	ResourceInfo: config.ResourceInfo{
		Name: "local-http",
		Type: config.TypeIngress,
	},
	Destination: config.Traffic{
		Driver: "k8s",

		Config: config.TrafficConfig{
			Cluster: "k8s_cluster.test",
			Port:    "1234",
			Address: "localhost",
		},
	},

	Source: config.Traffic{
		Driver: "local",

		Config: config.TrafficConfig{
			Port: "12344",
		},
	},
}
