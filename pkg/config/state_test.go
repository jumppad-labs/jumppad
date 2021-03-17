package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/shipyard-run/shipyard/pkg/utils"
	assert "github.com/stretchr/testify/require"
)

func setupConfigTests(t *testing.T) (*Config, func()) {
	dir := t.TempDir()

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	os.MkdirAll(utils.StateDir(), os.ModePerm)

	// create a config with all resource types
	c := New()

	// add the image cache
	cache := NewImageCache("docker-cache")
	cache.DependsOn = []string{"network.config"}
	c.AddResource(cache)

	con := NewContainer("config")
	con.Info().Module = "tester"
	c.AddResource(con)

	i := NewIngress("config")
	i.Id = "myid"
	c.AddResource(i)

	c.AddResource(NewDocs("config"))
	c.AddResource(NewExecLocal("config"))
	c.AddResource(NewExecRemote("config"))
	c.AddResource(NewHelm("config"))
	c.AddResource(NewK8sCluster("config"))
	c.AddResource(NewNetwork("config"))
	c.AddResource(NewNomadCluster("config"))

	return c, func() {
		os.Setenv("HOME", home)
	}
}

func TestConfigSerializesToJSON(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	statePath := utils.StatePath()
	err := c.ToJSON(statePath)

	assert.NoError(t, err)

	// check the file
	c2 := New()
	d, err := ioutil.ReadFile(statePath)
	assert.NoError(t, err)

	fmt.Println(string(d))
	err = json.Unmarshal(d, c2)
	assert.NoError(t, err)
	assert.Len(t, c2.Resources, c.ResourceCount())
}

func TestConfigDeSerializesFromJSON(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	err := ioutil.WriteFile(utils.StatePath(), []byte(complexState), os.ModePerm)
	assert.NoError(t, err)

	c = New()
	err = c.FromJSON(utils.StatePath())
	assert.NoError(t, err)

	assert.Len(t, c.Resources, 16)

	// check image cache
	r, err := c.FindResource("image_cache.docker-cache")
	assert.NoError(t, err)
	assert.NotNil(t, r)
}

func TestConfigMergesAddingItems(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c2 := New()
	c2.AddResource(NewContainer("test"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 11)
}

func TestConfigMergesWithExistingItemSetsPendingUpdateWhenApplied(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[1].Info().Status = Applied

	c2 := New()
	c2.AddResource(NewContainer("config"))

	cacheNew := NewImageCache("docker-cache")
	c2.AddResource(cacheNew)

	c.Merge(c2)

	assert.Len(t, c.Resources, 10)
	assert.Equal(t, c.Resources[1].Info().Status, PendingUpdate)
}

func TestConfigMergesWithExistingItemSetsItemCacheToPendingCreationWhenApplied(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[1].Info().Status = Applied

	c2 := New()
	c2.AddResource(NewContainer("config"))

	cacheNew := NewImageCache("docker-cache")
	c2.AddResource(cacheNew)

	c.Merge(c2)

	assert.Len(t, c.Resources, 10)
	assert.Equal(t, c.Resources[0].Info().Status, PendingCreation)
}

func TestConfigMergesWithExistingItemDoesNOTSetsPendingUpdateWhenOtherStatus(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[0].Info().Status = PendingCreation

	c2 := New()
	c2.AddResource(NewContainer("config"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 10)
	assert.Equal(t, c.Resources[0].Info().Status, PendingCreation)
}

func TestConfigMergesWithExistingItemRetainsStateFields(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[0].Info().Status = Applied

	c2 := New()
	c2.AddResource(NewIngress("config"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 10)
	assert.Equal(t, "myid", c.Resources[2].(*Ingress).Id)
}

func TestConfigMergesWithExistingItemAppendsDependencyOnCache(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[0].Info().Status = Applied

	c2 := New()
	c2.AddResource(NewNetwork("new"))

	cacheNew := NewImageCache("docker-cache")
	cacheNew.DependsOn = []string{"network.new"}
	c2.AddResource(cacheNew)

	c.Merge(c2)

	cache, _ := c.FindResource("image_cache.docker-cache")
	assert.Len(t, cache.Info().DependsOn, 2)
}

var complexState = `
{
  "blueprint": null,
  "resources": [
    {
      "depends_on": [
        "network.cloud",
        "network.onprem"
      ],
      "name": "docker-cache",
      "networks": [
        "network.cloud",
        "network.onprem"
      ],
      "status": "applied",
      "type": "image_cache"
    },
    {
      "chart": "/home/nicj/.shipyard/blueprints/github.com/shipyard-run/shipyard/examples/single_k3s_cluster/ref/testing/helm/consul-helm-0.22.0",
      "cluster": "k8s_cluster.k3s",
      "depends_on": [
        "module.consul",
        "k8s_cluster.k3s"
      ],
      "health_check": {
        "pods": [
          "release=consul"
        ],
        "timeout": "60s"
      },
      "module": "k8s",
      "name": "consul",
      "namespace": "default",
      "status": "applied",
      "type": "helm",
      "values": "/home/nicj/.shipyard/blueprints/github.com/shipyard-run/shipyard/examples/single_k3s_cluster/ref/testing/helm/consul-values.yaml",
      "values_string": null
    },
    {
      "chart": "/home/nicj/.shipyard/helm_charts/github.com/hashicorp/vault-helm",
      "cluster": "k8s_cluster.k3s",
      "depends_on": [
        "module.consul",
        "k8s_cluster.k3s"
      ],
      "health_check": {
        "pods": [
          "app.kubernetes.io/name=vault"
        ],
        "timeout": "120s"
      },
      "module": "k8s",
      "name": "vault",
      "namespace": "default",
      "status": "applied",
      "type": "helm",
      "values": "",
      "values_string": null
    },
    {
      "cluster": "k8s_cluster.k3s",
      "depends_on": [
        "module.consul",
        "network.cloud",
        "k8s_cluster.k3s"
      ],
      "module": "k8s",
      "name": "consul-http",
      "networks": [
        {
          "name": "network.cloud"
        }
      ],
      "ports": [
        {
          "host": "18500",
          "local": "8500",
          "open_in_browser": "",
          "remote": "8500"
        }
      ],
      "service": "consul-consul-server",
      "status": "applied",
      "type": "k8s_ingress"
    },
    {
      "depends_on": [
        "module.consul",
        "k8s_cluster.k3s"
      ],
      "destination": {
        "config": {
          "address": "consul-consul-server.default.svc",
          "cluster": "k8s_cluster.k3s",
          "port": "8300"
        },
        "driver": "k8s"
      },
      "id": "db729439-5bef-48da-9d5a-089d54d3dc83",
      "module": "k8s",
      "name": "consul-lan",
      "source": {
        "config": {
          "port": "8300"
        },
        "driver": "local"
      },
      "status": "applied",
      "type": "ingress"
    },
    {
      "cluster": "k8s_cluster.k3s",
      "depends_on": [
        "module.consul",
        "network.cloud",
        "k8s_cluster.k3s"
      ],
      "module": "k8s",
      "name": "vault-http",
      "networks": [
        {
          "name": "network.cloud"
        }
      ],
      "ports": [
        {
          "host": "18200",
          "local": "8200",
          "open_in_browser": "",
          "remote": "8200"
        }
      ],
      "service": "vault",
      "status": "applied",
      "type": "k8s_ingress"
    },
    {
      "depends_on": [
        "module.consul",
        "network.cloud"
      ],
      "driver": "k3s",
      "images": [
        {
          "name": "shipyardrun/connector:v0.0.10"
        }
      ],
      "module": "k8s",
      "name": "k3s",
      "networks": [
        {
          "name": "network.cloud"
        }
      ],
      "nodes": 1,
      "status": "applied",
      "type": "k8s_cluster",
      "version": "v1.18.16"
    },
    {
      "depends_on": [
        "module.consul"
      ],
      "module": "k8s",
      "name": "cloud",
      "status": "applied",
      "subnet": "10.5.0.0/16",
      "type": "network"
    },
    {
      "destination": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config/consul.hcl",
      "module": "consul",
      "name": "consul_config",
      "source": "data_dir = \"#{{ .Vars.data_dir }}\"\nlog_level = \"DEBUG\"\n\ndatacenter = \"dc1\"\nprimary_datacenter = \"dc1\"\n\nserver = true\n\nbootstrap_expect = 1\nui = true\n\nbind_addr = \"0.0.0.0\"\nclient_addr = \"0.0.0.0\"\nadvertise_addr = \"10.6.0.200\"\n\nports {\n  grpc = 8502\n}\n\nconnect {\n  enabled = true\n}\n",
      "status": "applied",
      "type": "template",
      "vars": {
        "data_dir": "/tmp"
      }
    },
    {
      "build": null,
      "command": [
        "consul",
        "agent",
        "-config-file=/config/consul.hcl"
      ],
      "depends": [
        "template.consul_config"
      ],
      "depends_on": [
        "network.onprem",
        "template.consul_config"
      ],
      "environment": [
        {
          "key": "something",
          "value": "this is a module"
        },
        {
          "key": "foo",
          "value": ""
        },
        {
          "key": "file",
          "value": "this is the contents of a file"
        },
        {
          "key": "abc",
          "value": "123"
        },
        {
          "key": "SHIPYARD_FOLDER",
          "value": "/home/nicj/.shipyard"
        },
        {
          "key": "HOME_FOLDER",
          "value": "/home/nicj"
        }
      ],
      "image": {
        "name": "consul:1.8.1"
      },
      "module": "consul",
      "name": "consul",
      "networks": [
        {
          "aliases": [
            "myalias"
          ],
          "name": "network.onprem"
        }
      ],
      "resources": {
        "cpu": 2000,
        "memory": 1024
      },
      "status": "applied",
      "type": "container",
      "volumes": [
        {
          "destination": "/config",
          "source": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config"
        }
      ]
    },
    {
      "command": [
        "tail",
        "-f",
        "/dev/null"
      ],
      "depends_on": [
        "container.consul"
      ],
      "image": {
        "name": "envoyproxy/envoy-alpine:v1.14.3"
      },
      "module": "consul",
      "name": "envoy",
      "status": "applied",
      "target": "container.consul",
      "type": "sidecar",
      "volumes": [
        {
          "destination": "/config",
          "source": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config"
        }
      ]
    },
    {
      "depends_on": [
        "network.onprem",
        "container.consul"
      ],
      "module": "consul",
      "name": "consul-container-http",
      "networks": [
        {
          "name": "network.onprem"
        }
      ],
      "ports": [
        {
          "host": "28500",
          "local": "8500",
          "open_in_browser": "",
          "remote": "8500"
        }
      ],
      "status": "applied",
      "target": "container.consul",
      "type": "container_ingress"
    },
    {
      "module": "consul",
      "name": "onprem",
      "status": "applied",
      "subnet": "10.6.0.0/16",
      "type": "network"
    },
    {
      "module": "consul",
      "name": "consul_http_addr",
      "status": "applied",
      "type": "output",
      "value": "http://consul.container.shipyard.run:8500"
    },
    {
      "depends_on": [
        "network.onprem",
        "container.consul"
      ],
      "name": "consul-container-http-2",
      "networks": [
        {
          "name": "network.onprem"
        }
      ],
      "ports": [
        {
          "host": "18600",
          "local": "8500",
          "open_in_browser": "",
          "remote": "8500"
        }
      ],
      "status": "applied",
      "target": "container.consul",
      "type": "container_ingress"
    },
    {
      "depends_on": [
        "container_ingress.consul-container-http-2"
      ],
      "index_pages": [
        "index",
        "other"
      ],
      "index_title": "Test",
      "module": "docs",
      "name": "docs",
      "open_in_browser": true,
      "path": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/docs/docs",
      "port": 8080,
      "status": "applied",
      "type": "docs"
    }
  ]
}
`
