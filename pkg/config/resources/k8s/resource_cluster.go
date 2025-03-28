package k8s

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeK8sCluster is the resource string for a Cluster resource
const TypeK8sCluster string = "k8s_cluster"
const TypeKubernetesCluster string = "kubernetes_cluster"

/*
The `kubernetes_cluster` resource allows you to create immutable Kubernetes clusters running in Docker containers using K3s.

```hcl

	resource "kubernetes_cluster" "name" {
	  ...
	}

```

## Image Caching

Kubernetes clusters do not share the local machines Docker image cache. Each node in a cluster has it's own unqiue cache.

To save bandwidth all containers launched in the Kubernetes cluster pulled through an image cache that runs in Docker.
After the first pull all images are subsequently pulled from the image cache not the public internet.
This cache is global to all Nomad and Kubernetes clusters created with Jumppad.

For more information on the image cache see the `container_registry` resource.

@include container.Image
@include container.NetworkAttachment
@include container.Port
@include container.PortRange
@include container.Volume
@include k8s.ClusterConfig
@include k8s.DockerConfig
@include k8s.KubeConfig

@resource

@example Simple cluster
```

	resource "network" "cloud" {
	  subnet = "10.5.0.0/16"
	}

	resource "kubernetes_cluster" "cluster" {
	  network {
	    id = resource.network.cloud.meta.id
	  }
	}

	output "KUBECONFIG" {
	  value = resource.kubernetes_cluster.cluster.kube_config.path
	}

```

@example Full Example
```

	resource "network" "cloud" {
	  subnet = "10.5.0.0/16"
	}

	resource "kubernetes_cluster" "cluster" {
	  network {
	    id = resource.network.cloud.meta.id
	  }

	  copy_image {
	    name = "shipyardrun/connector:v0.1.0"
	  }
	}

	resource "k8s_config" "fake_service" {
	  cluster = resource.kubernetes_cluster.cluster

	  paths = ["./fake_service.yaml"]

	  health_check {
	    timeout = "240s"
	    pods    = ["app.kubernetes.io/name=fake-service"]
	  }
	}

	resource "helm" "vault" {
	  cluster = resource.kubernetes_cluster.cluster

	  repository {
	    name = "hashicorp"
	    url  = "https://helm.releases.hashicorp.com"
	  }

	  chart   = "hashicorp/vault"
	  version = "v0.18.0"

	  values = "./helm/vault-values.yaml"

	  health_check {
	    timeout = "240s"
	    pods    = ["app.kubernetes.io/name=vault"]
	  }
	}

	resource "ingress" "vault_http" {
	  port = 18200

	  target {
	    resource = resource.kubernetes_cluster.cluster
	    port = 8200

	    config = {
	      service   = "vault"
	      namespace = "default"
	    }
	  }
	}

	resource "ingress" "fake_service" {
	  port = 19090

	  target {
	    resource = resource.kubernetes_cluster.cluster
	    port = 9090

	    config = {
	      service   = "fake-service"
	      namespace = "default"
	    }
	  }
	}

	output "VAULT_ADDR" {
	  value = "http://${resource.ingress.vault_http.local_address}"
	}

	output "KUBECONFIG" {
	  value = resource.kubernetes_cluster.cluster.kube_config.path
	}

```
*/
type Cluster struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		Network attaches the container to an existing network defined in a separate stanza.
		This block can be specified multiple times to attach the container to multiple networks.

		```hcl
		network {
		  id = resource.network.main.meta.id
		}
		```

		@type []NetworkAttachment
	*/
	Networks []container.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified
	/*
		Image defines a Docker image to use when creating the container.
		By default the kubernetes cluster resource will be created using the latest Jumppad container image.

		```hcl
		image {
		  name = "example/kubernetes:latest"
		}
		```

		@type Image
	*/
	Image *container.Image `hcl:"image,block" json:"images,omitempty"`
	/*
		The number of nodes to create in the cluster.

		```hcl
		nodes = 3
		```
	*/
	Nodes int `hcl:"nodes,optional" json:"nodes,omitempty"`
	/*
		Additional volume to mount to the server and client nodes.

		```hcl
		volume {
		  source = "./mydirectory"
		  destination = "/path_in_container"
		}
		```

		@type []Volume
	*/
	Volumes []container.Volume `hcl:"volume,block" json:"volumes,omitempty"`
	/*
		Docker image in the local Docker image cache to copy to the cluster on creation.
		This image is added to the Kubernetes clients docker cache enabling jobs to use images that may not be in the local registry.

		Jumppad tracks changes to copied images, should the image change running jumppad up would push any changes to the cluster automatically.

		```hcl
		copy_image {
		  name = "mylocalimage:version"
		}
		```

		@type []Image
	*/
	CopyImages []container.Image `hcl:"copy_image,block" json:"copy_images,omitempty"`
	/*
		A `port` stanza allows you to expose container ports on the local network or host.
		This stanza can be specified multiple times.

		```hcl
		port {
		  local = 80
		  host  = 8080
		}
		```

		@type []Port
	*/
	Ports []container.Port `hcl:"port,block" json:"ports,omitempty"`
	/*
		A `port_range` stanza allows you to expose a range of container ports on the local network or host.
		This stanza can be specified multiple times.

		The following example would create 11 ports from 80 to 90 (inclusive) and expose them to the host machine.

		```hcl
		port {
		  range       = "80-90"
		  enable_host = true
		}
		```

		@type []PortRange
	*/
	PortRanges []container.PortRange `hcl:"port_range,block" json:"port_ranges,omitempty"`
	/*
		An env stanza allows you to set environment variables in the container. This stanza can be specified multiple times.

		```hcl
		env {
		  key   = "PATH"
		  value = "/usr/local/bin"
		}
		```
	*/
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"` // environment variables to set when starting the container
	/*
		Specifies the configuration for the Kubernetes cluster.
	*/
	Config *ClusterConfig `hcl:"config,block" json:"config,omitempty"`
	/*
		Details for the Kubenetes config file that can be used to interact with the cluster.

		@computed
	*/
	KubeConfig KubeConfig `hcl:"kube_config,optional" json:"kube_config,omitempty"`
	/*
		Port to expose the Kubernetes API on the host.
		By default this uses the standard api port `443`; however, if you are running multiple kubernetes instances you will need to override this value.
	*/
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`
	/*
		The port where the Jumppad connector is exposed to the host, this property is requied by the ingress resource and is
		not generally needed when building blueprints.

		@computed
	*/
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`
	/*
		The fully qualified resource name for the Kubernetes cluster, this value can be used to address the server from the Docker network.
		It is also the name of the Docker container.

		@example
		```
		server.name.k8s-cluster.local.jmpd.in
		````
	*/
	ContainerName string `hcl:"container_name,optional" json:"container_name,omitempty"`
	/*
		Local IP address of the Nomad server, this property can be used to set the NOAMD_ADDR on the Jumppad client.

		@example
		```
		output "K8S_ADDR" {
		value = "https://${resource.kubernetes_cluster.dev.external_ip}:${resource.kubernetes_cluster.dev.api_port}"
		}
		```

		@computed
	*/
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

/*
Specifies the configuration for the Kubernetes cluster.

```hcl

	resource "kubernetes_cluster" "cluster" {
	  config {
	    ...
	  }
	}

```
*/
type ClusterConfig struct {
	/*
		Docker configuration for the Kubernetes cluster.

		```hcl
		docker {
		  no_proxy            = ["insecure.container.local.jmpd.in"]
		  insecure_registries = ["insecure.container.local.jmpd.in:5003"]
		}
		````
	*/
	DockerConfig *DockerConfig `hcl:"docker,block" json:"docker,omitempty"`
}

/*
Specifies the configuration for the Docker engine in the cluster.

```hcl

	resource "kubernetes_cluster" "cluster" {
	  config {
	    docker {
	      ...
	    }
	  }
	}

```
*/
type DockerConfig struct {
	/*
		A list of docker registries that should not be proxied.

		```hcl
		no_proxy = ["insecure.container.local.jmpd.in"]
		```
	*/
	NoProxy []string `hcl:"no_proxy,optional" json:"no-proxy,omitempty"`
	/*
		A list of insecure docker registries.

		```hcl
		insecure_registries = ["insecure.container.local.jmpd.in:5003"]
		```
	*/
	InsecureRegistries []string `hcl:"insecure_registries,optional" json:"insecure-registries,omitempty"`
}

/*
Details for the Kubenetes config file that can be used to interact with the cluster.
*/
type KubeConfig struct {
	/*
		The path to the kubeconfig file

		@computed
	*/
	ConfigPath string `hcl:"path" json:"path"`
	/*
		The base64 encoded ca certificate

		@computed
	*/
	CA string `hcl:"ca" json:"ca"`
	/*
		The base64 encoded client certificate

		@computed
	*/
	ClientCertificate string `hcl:"client_certificate" json:"client_certificate"`
	/*
		The base64 encoded client key

		@computed
	*/
	ClientKey string `hcl:"client_key" json:"client_key"`
}

const k3sBaseImage = "ghcr.io/jumppad-labs/kubernetes"
const k3sBaseVersion = "v1.31.1"

func (k *Cluster) Process() error {
	if k.APIPort == 0 {
		k.APIPort = 443
	}

	if k.Image == nil {
		k.Image = &container.Image{Name: fmt.Sprintf("%s:%s", k3sBaseImage, k3sBaseVersion)}
	}

	for i, v := range k.Volumes {
		k.Volumes[i].Source = utils.EnsureAbsolute(v.Source, k.Meta.File)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(k.Meta.ID)
		if r != nil {
			kstate := r.(*Cluster)
			k.KubeConfig = kstate.KubeConfig
			k.ContainerName = kstate.ContainerName
			k.APIPort = kstate.APIPort
			k.ConnectorPort = kstate.ConnectorPort
			k.ExternalIP = kstate.ExternalIP
			k.KubeConfig = kstate.KubeConfig

			// add the network addresses
			for _, a := range kstate.Networks {
				for i, m := range k.Networks {
					if m.ID == a.ID {
						k.Networks[i].IPAddress = a.IPAddress
						k.Networks[i].Name = a.Name
						break
					}
				}
			}

			// add the image id from state
			for x, img := range k.CopyImages {
				for _, sImg := range kstate.CopyImages {
					if img.Name == sImg.Name && img.Username == sImg.Username {
						k.CopyImages[x].ID = sImg.ID
					}
				}
			}

			// the network name is set
			copy(k.Networks, kstate.Networks)
		}
	}

	return nil
}
