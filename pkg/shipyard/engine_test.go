package shipyard

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/providers/mocks"
	"github.com/shipyard-run/shipyard/pkg/utils"

	assert "github.com/stretchr/testify/require"
)

var lock = sync.Mutex{}

func setupTests(returnVals map[string]error) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	return setupTestsBase(returnVals, "")
}

func setupTestsWithState(returnVals map[string]error, state string) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	return setupTestsBase(returnVals, state)
}

func setupState(state string) func() {
	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), dir)

	// write the state file
	if state != "" {
		os.MkdirAll(utils.StateDir(), os.ModePerm)
		f, err := os.Create(utils.StatePath())
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.WriteString(state)
		if err != nil {
			panic(err)
		}
	}

	return func() {
		os.Setenv(utils.HomeEnvName(), home)
		os.RemoveAll(dir)
	}
}

func setupTestsBase(returnVals map[string]error, state string) (Engine, *config.Config, *[]*mocks.MockProvider, func()) {
	log.SetOutput(ioutil.Discard)

	p := &[]*mocks.MockProvider{}

	cl := &Clients{}
	e := &EngineImpl{
		clients:     cl,
		log:         hclog.NewNullLogger(),
		getProvider: generateProviderMock(p, returnVals),
	}

	return e, nil, p, setupState(state)
}

func generateProviderMock(mp *[]*mocks.MockProvider, returnVals map[string]error) getProviderFunc {
	return func(c config.Resource, cc *Clients) providers.Provider {
		lock.Lock()
		defer lock.Unlock()

		m := mocks.New(c)

		val := returnVals[c.Info().Name]
		m.On("Create").Return(val)
		m.On("Destroy").Return(val)

		*mp = append(*mp, m)
		return m
	}
}

func getTestFiles(tests string) string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path := path.Dir(e)
	return filepath.Join(path, "/examples", tests)
}

func TestNewCreatesClients(t *testing.T) {
	e, err := New(hclog.NewNullLogger())
	assert.NoError(t, err)

	cl := e.GetClients()

	assert.NotNil(t, cl.Kubernetes)
	assert.NotNil(t, cl.Helm)
	assert.NotNil(t, cl.Command)
	assert.NotNil(t, cl.HTTP)
	assert.NotNil(t, cl.Nomad)
	assert.NotNil(t, cl.Getter)
	assert.NotNil(t, cl.Browser)
	assert.NotNil(t, cl.ImageLog)
	assert.NotNil(t, cl.Connector)
}

func TestApplyWithSingleFile(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_file/container.hcl")
	assert.NoError(t, err)

	assert.Equal(t, "onprem", (*mp)[0].Config().Info().Name)

	// can either be consul or the image cache
	assert.Contains(t, []string{"consul", "docker-cache"}, (*mp)[2].Config().Info().Name)
}

func TestApplyAddsImageCache(t *testing.T) {
	e, _, _, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_file/container.hcl")
	assert.NoError(t, err)

	dc := e.ResourceCountForType(string(config.TypeImageCache))
	assert.Equal(t, 1, dc)
}

func TestApplyWithSingleFileAndVariables(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.ApplyWithVariables("../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	assert.NoError(t, err)

	assert.Equal(t, "onprem", (*mp)[0].Config().Info().Name)

	assert.Contains(t, []string{"consul", "docker-cache", "local_connector"}, (*mp)[1].Config().Info().Name)
}

func TestApplyCallsProviderInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.NoError(t, err)

	// should have called in order
	assert.Equal(t, "cloud", (*mp)[0].Config().Info().Name)

	// due to paralel nature of the DAG, these two elements will be first
	assert.Contains(t, []string{"docker-cache", "k3s", "local_connector"}, (*mp)[1].Config().Info().Name)

	// due to paralel nature of the DAG, these two elements can appear in any order
	assert.Contains(t, []string{"docker-cache", "consul-http", "consul", "vault", "vault-http", "consul-lan", "k3s"}, (*mp)[2].Config().Info().Name)
	assert.Contains(t, []string{"docker-cache", "consul-http", "consul", "vault", "vault-http", "consul-lan", "k3s"}, (*mp)[3].Config().Info().Name)
	assert.Contains(t, []string{"docker-cache", "consul-http", "consul", "vault", "vault-http", "consul-lan", "k3s"}, (*mp)[4].Config().Info().Name)
	assert.Contains(t, []string{"docker-cache", "consul-http", "consul", "vault", "vault-http", "consul-lan", "k3s"}, (*mp)[5].Config().Info().Name)
	assert.Contains(t, []string{"docker-cache", "consul-http", "consul", "vault", "vault-http", "consul-lan", "k3s"}, (*mp)[6].Config().Info().Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 8)
	//assert.Len(t, res, 4)
}

func TestApplyCallsProviderDestroyForResourcesPendingorFailed(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(nil, failedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created
}

func TestApplyReturnsErrorWhenProviderDestroyForResourcesPendingorFailed(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(map[string]error{"dc1": fmt.Errorf("boom")}, failedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created
}

func TestApplyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplyCallsProviderCreateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	_, err := e.Apply("../../examples/single_k3s_cluster")
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1)
}

func TestApplySetsStatusForEachResource(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(nil, mergedState)
	defer cleanup()

	_, err := e.Apply("")
	assert.NoError(t, err)

	// should only call create and destroy for the cache as this is pending update
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, _, mp, cleanup := setupTests(nil)
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 8)
}

func TestDestroyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"k3s": fmt.Errorf("boom")})
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 7)
}

func TestDestroyFailSetsStatus(t *testing.T) {
	e, _, mp, cleanup := setupTests(map[string]error{"cloud": fmt.Errorf("boom")})
	defer cleanup()

	err := e.Destroy("../../examples/single_k3s_cluster", true)
	assert.Error(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 8)
	assert.Equal(t, config.Failed, (*mp)[7].Config().Info().Status)
}

func TestDestroyCallsProviderDestroyInCorrectOrder(t *testing.T) {
	e, _, mp, cleanup := setupTestsWithState(nil, complexState)
	defer cleanup()

	err := e.Destroy("", true)
	assert.NoError(t, err)

	// network should be last to be removed
	assert.Equal(t, "cloud", (*mp)[7].Config().Info().Name)
}

func testAssertMethodCalled(t *testing.T, p *[]*mocks.MockProvider, method string, n int, args ...interface{}) {
	callCount := 0

	for _, pm := range *p {
		// cast the provider into a mock
		for _, c := range pm.Calls {
			if c.Method == method {
				callCount++
			}
		}
	}

	if callCount != n {
		t.Fatalf("Expected %d calls got %d", n, callCount)
	}
}

var failedState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "failed",
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var mergedState = `
{
  "blueprint": null,
  "resources": [
	{
      "name": "dc1",
      "status": "pending_update",
      "subnet": "10.15.0.0/16",
      "type": "network"
	}
  ]
}
`

var complexState = `
{
  "blueprint": {
    "title": "Consul Service Mesh on Kubernetes with Monitoring",
    "author": "Nic Jackson",
    "slug": "k8s_consul_stack",
    "intro": "# Consul Service Mesh on Kubernetes with Monitoring\n\nThis blueprint creates a Kubernetes cluster and installs the following elements:\n\n* Consul Service Mesh With CRDs\n* Prometheus\n* Loki\n* Grafana\n\nTo access Grafana the following details can be used:\n\n* user: admin\n* pass: admin\n\nACLs are disabled for Consul",
    "health_check_timeout": "30s"
  },
  "resources": [
    {
      "name": "docker-cache",
      "type": "image_cache",
      "status": "applied",
      "depends_on": [
        "network.docs",
        "network.dc1"
      ],
      "networks": [
        "network.docs",
        "network.dc1"
      ]
    },
    {
      "name": "consul",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1"
      ],
      "module": "consul",
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/hashicorp/consul-helm/ref/v0.28.0",
      "values": "/home/nicj/.shipyard/blueprints/github.com/nicholasjackson/hashicorp-shipyard-modules/modules/consul/helm/consul_values.yaml",
      "values_string": null,
      "chart_name": "consul",
      "namespace": "default",
      "health_check": {
        "timeout": "60s",
        "pods": [
          "app=consul"
        ]
      }
    },
    {
      "name": "consul",
      "type": "ingress",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1"
      ],
      "module": "consul",
      "id": "902a4f51-6b8f-4159-8e72-36d3ff89076d",
      "destination": {
        "driver": "k8s",
        "config": {
          "cluster": "k8s_cluster.dc1",
          "address": "consul-server.default.svc",
          "port": "8500"
        }
      },
      "source": {
        "driver": "local",
        "config": {
          "port": "8500"
        }
      }
    },
    {
      "name": "consul-rpc",
      "type": "ingress",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1"
      ],
      "module": "consul",
      "id": "766be91e-304e-4b95-bd1d-cfe79124acac",
      "destination": {
        "driver": "k8s",
        "config": {
          "cluster": "k8s_cluster.dc1",
          "address": "consul-server.default.svc",
          "port": "8300"
        }
      },
      "source": {
        "driver": "local",
        "config": {
          "port": "8300"
        }
      }
    },
    {
      "name": "consul-lan-serf",
      "type": "ingress",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1"
      ],
      "module": "consul",
      "id": "b6ea03b8-3e4f-417a-ac65-5ab83e927989",
      "destination": {
        "driver": "k8s",
        "config": {
          "cluster": "k8s_cluster.dc1",
          "address": "consul-server.default.svc",
          "port": "8301"
        }
      },
      "source": {
        "driver": "local",
        "config": {
          "port": "8301"
        }
      }
    },
    {
      "name": "CONSUL_HTTP_ADDR",
      "type": "output",
      "status": "applied",
      "module": "consul",
      "value": "127.0.0.1:8500"
    },
    {
      "name": "cert-manager",
      "type": "k8s_config",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "smi_controller",
      "cluster": "k8s_cluster.dc1",
      "paths": [
        "/home/nicj/go/src/github.com/nicholasjackson/hashicorp-blueprints/modules/smi-controller/cert-manager.crds.yaml"
      ],
      "wait_until_ready": true
    },
    {
      "name": "cert-manager",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1",
        "k8s_config.cert-manager"
      ],
      "module": "smi_controller",
      "depends": [
        "k8s_config.cert-manager"
      ],
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/jetstack/cert-manager/ref/v1.2.0/deploy/charts/cert-manager",
      "values": "/home/nicj/go/src/github.com/nicholasjackson/hashicorp-blueprints/modules/smi-controller/helm/cert-manager-helm-values.yaml",
      "values_string": null,
      "chart_name": "cert-manager",
      "namespace": "smi",
      "create_namespace": true,
      "health_check": {
        "timeout": "60s",
        "pods": [
          "app.kubernetes.io/instance=cert-manager"
        ]
      }
    },
    {
      "name": "smi_controller_config",
      "type": "template",
      "status": "applied",
      "depends_on": [
        "module.consul"
      ],
      "module": "smi_controller",
      "source": "controller:\n  enabled: \"true\"\n\n  image:\n    repository: \"nicholasjackson/smi-controller-example\"\n    pullPolicy: IfNotPresent\n    # Overrides the image tag whose default is the chart appVersion.\n    tag: \"dev\"\n",
      "destination": "/home/nicj/go/src/github.com/nicholasjackson/hashicorp-blueprints/modules/smi-controller/helm/smi-controller-values.yaml"
    },
    {
      "name": "smi-controler",
      "type": "helm",
      "status": "failed",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1",
        "helm.cert-manager",
        "template.smi_controller_config"
      ],
      "module": "smi_controller",
      "depends": [
        "helm.cert-manager",
        "template.smi_controller_config"
      ],
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/nicholasjackson/smi-controller-sdk/helm/smi-controller",
      "values": "/home/nicj/go/src/github.com/nicholasjackson/consul-canary-deployment/shipyard/../helm/consul-smi-controller.yaml",
      "values_string": null,
      "chart_name": "smi-controler",
      "namespace": "smi"
    },
    {
      "name": "docs",
      "type": "network",
      "status": "applied",
      "module": "docs",
      "subnet": "10.6.0.0/16"
    },
    {
      "name": "docs",
      "type": "docs",
      "status": "applied",
      "depends_on": [
        "network.dc1"
      ],
      "module": "docs",
      "networks": [
        {
          "name": "network.dc1"
        }
      ],
      "path": "/home/nicj/go/src/github.com/nicholasjackson/consul-canary-deployment/shipyard/docs/pages",
      "port": 18080,
      "live_reload_port": 37950,
      "open_in_browser": true,
      "index_title": "Canary_Deployments",
      "index_pages": [
        "index",
        "configuration",
        "flagger",
        "application",
        "load_generation",
        "grafana",
        "canary",
        "rollback",
        "summary"
      ]
    },
    {
      "name": "tools",
      "type": "container",
      "status": "applied",
      "depends_on": [
        "network.dc1"
      ],
      "networks": [
        {
          "name": "network.dc1"
        }
      ],
      "image": {
        "name": "shipyardrun/tools:latest"
      },
      "build": null,
      "command": [
        "tail",
        "-f",
        "/dev/null"
      ],
      "environment": [
        {
          "key": "KUBECONFIG",
          "value": "/root/.config/kubeconfig.yaml"
        },
        {
          "key": "HOST",
          "value": "172.25.154.69"
        },
        {
          "key": "CONSUL_HTTP_ADDR",
          "value": "http://172.25.154.69:8500"
        }
      ],
      "volumes": [
        {
          "source": "/home/nicj/go/src/github.com/nicholasjackson/app",
          "destination": "/app"
        },
        {
          "source": "/home/nicj/.shipyard/config/dc1/kubeconfig-docker.yaml",
          "destination": "/root/.config/kubeconfig.yaml"
        }
      ]
    },
    {
      "name": "flagger",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1"
      ],
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/fluxcd/flagger/charts/flagger",
      "values": "/home/nicj/go/src/github.com/nicholasjackson/consul-canary-deployment/shipyard/../helm/flagger-values.yaml",
      "values_string": null,
      "chart_name": "flagger",
      "namespace": "default"
    },
    {
      "name": "dc1",
      "type": "k8s_cluster",
      "status": "applied",
      "depends_on": [
        "network.dc1"
      ],
      "networks": [
        {
          "name": "network.dc1"
        }
      ],
      "driver": "k3s",
      "version": "v1.18.16",
      "nodes": 1
    },
    {
      "name": "grafana_secret",
      "type": "k8s_config",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "cluster": "k8s_cluster.dc1",
      "paths": [
        "/home/nicj/.shipyard/blueprints/github.com/nicholasjackson/hashicorp-shipyard-modules/modules/monitoring/k8sconfig/grafana_secret.yaml"
      ],
      "wait_until_ready": true
    },
    {
      "name": "grafana",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/grafana/helm-charts/charts/grafana",
      "values": "/home/nicj/go/src/github.com/nicholasjackson/consul-canary-deployment/shipyard/../helm/grafana-values.yaml",
      "values_string": {
        "admin.existingSecret": "grafana-password"
      },
      "chart_name": "grafana",
      "namespace": "default"
    },
    {
      "name": "grafana",
      "type": "ingress",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "id": "b286121a-5e9d-43ac-b91e-bfdbf613b096",
      "destination": {
        "driver": "k8s",
        "config": {
          "cluster": "k8s_cluster.dc1",
          "address": "grafana.default.svc",
          "port": "80"
        }
      },
      "source": {
        "driver": "local",
        "config": {
          "port": "8080"
        }
      }
    },
    {
      "name": "loki",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/grafana/helm-charts/charts/loki",
      "values": "",
      "values_string": null,
      "chart_name": "loki",
      "namespace": "default"
    },
    {
      "name": "promtail",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/grafana/helm-charts/charts/promtail",
      "values": "",
      "values_string": {
        "grafana.enabled": "false",
        "loki.serviceName": "loki"
      },
      "chart_name": "promtail",
      "namespace": "default"
    },
    {
      "name": "GRAFANA_HTTP_ADDR",
      "type": "output",
      "status": "applied",
      "depends_on": [
        "module.consul"
      ],
      "module": "monitoring",
      "value": "127.0.0.1:8080"
    },
    {
      "name": "PROMETHEUS_HTTP_ADDR",
      "type": "output",
      "status": "applied",
      "depends_on": [
        "module.consul"
      ],
      "module": "monitoring",
      "value": "127.0.0.1:9090"
    },
    {
      "name": "GRAFANA_USER",
      "type": "output",
      "status": "applied",
      "depends_on": [
        "module.consul"
      ],
      "module": "monitoring",
      "value": "admin"
    },
    {
      "name": "GRAFANA_PASWORD",
      "type": "output",
      "status": "applied",
      "depends_on": [
        "module.consul"
      ],
      "module": "monitoring",
      "value": "admin"
    },
    {
      "name": "prometheus",
      "type": "helm",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "cluster": "k8s_cluster.dc1",
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/prometheus-community/helm-charts/charts/kube-prometheus-stack",
      "values": "",
      "values_string": {
        "alertmanager.enabled": "false",
        "grafana.enabled": "false"
      },
      "chart_name": "prometheus",
      "namespace": "default",
      "health_check": {
        "timeout": "90s",
        "pods": [
          "release=prometheus"
        ]
      }
    },
    {
      "name": "prometheus",
      "type": "k8s_config",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1",
        "helm.prometheus"
      ],
      "module": "monitoring",
      "depends": [
        "helm.prometheus"
      ],
      "cluster": "k8s_cluster.dc1",
      "paths": [
        "/home/nicj/.shipyard/blueprints/github.com/nicholasjackson/hashicorp-shipyard-modules/modules/monitoring/k8sconfig/prometheus_operator.yaml"
      ],
      "wait_until_ready": true
    },
    {
      "name": "prometheus",
      "type": "ingress",
      "status": "applied",
      "depends_on": [
        "module.consul",
        "k8s_cluster.dc1"
      ],
      "module": "monitoring",
      "id": "5615ef18-8658-415c-84a5-be9b7b51ed72",
      "destination": {
        "driver": "k8s",
        "config": {
          "cluster": "k8s_cluster.dc1",
          "address": "prometheus-kube-prom-prometheus.default.svc",
          "port": "9090"
        }
      },
      "source": {
        "driver": "local",
        "config": {
          "port": "9090"
        }
      }
    },
    {
      "name": "prometheus-setup",
      "type": "k8s_config",
      "status": "applied",
      "depends_on": [
        "k8s_cluster.dc1",
        "module.monitoring"
      ],
      "depends": [
        "module.monitoring"
      ],
      "cluster": "k8s_cluster.dc1",
      "paths": [
        "/home/nicj/go/src/github.com/nicholasjackson/consul-canary-deployment/shipyard/../setup/prometheus-config.yaml"
      ],
      "wait_until_ready": true
    },
    {
      "name": "dc1",
      "type": "network",
      "status": "applied",
      "subnet": "10.5.0.0/16"
    },
    {
      "name": "KUBECONFIG",
      "type": "output",
      "status": "applied",
      "value": "/home/nicj/.shipyard/config/dc1/kubeconfig.yaml"
    }
  ]
}
`
