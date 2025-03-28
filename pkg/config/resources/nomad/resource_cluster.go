package nomad

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster string = "nomad_cluster"

/*
The `nomad_cluster` resource allows you to create Nomad clusters as Docker containers.
Clusters can either be a single node combined server and client, or comprised of a dedicated server and client nodes.

```hcl

	resource "nomad_cluster" "name" {
	  ...
	}

```

## Image Caching

Nomad clusters do not share the local machines Docker image cache. Each node in a cluster has it's own unqiue cache.

To save bandwidth all containers launched in the Nomad cluster pulled through an image cache that runs in Docker.
After the first pull all images are subsequently pulled from the image cache not the public internet.
This cache is global to all Nomad and Kubernetes clusters created with Jumppad.

For more information on the image cache see the `container_registry` resource.

@include container.Image
@include container.NetworkAttachment
@include container.Port
@include container.PortRange
@include container.Volume
@include nomad.Config
@include nomad.DockerConfig

@resource

@example Minimal Example
```

	resource "network" "cloud" {
	  subnet = "10.10.0.0/16"
	}

	resource "nomad_cluster" "dev" {
	  network {
	    id = resource.network.cloud.meta.id
	  }
	}

```

@example Full Example
```

	resource "network" "cloud" {
	  subnet = "10.10.0.0/16"
	}

	resource "nomad_cluster" "dev" {
	  client_nodes=3

	  network {
	    id = resource.network.cloud.meta.id
	  }
	}

	resource "nomad_job" "example_1" {
	  cluster = resource.nomad_cluster.dev

	  paths = ["./app_config/example1.nomad"]

	  health_check {
	    timeout    = "60s"
	    nomad_jobs = ["example_1"]
	  }
	}

	resource "ingress" "fake_service_1" {
	  port = 19090

	  target {
	    resource   = resource.nomad_cluster.dev
	    named_port = "http"

	    config = {
	      job   = "example_1"
	      group = "fake_service"
	      task  = "fake_service"
	    }
	  }
	}

```
*/
type NomadCluster struct {
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
	Networks container.NetworkAttachments `hcl:"network,block" json:"networks,omitempty"`
	/*
		Image defines a Docker image to use when creating the container.
		By default the nomad cluster resource will be created using the latest container image.

		```hcl
		image {
		  name = "example/nomad:latest"
		}
		```

		@type Image
	*/
	Image *container.Image `hcl:"image,block" json:"images,omitempty"`
	/*
		Number of client nodes to create, if set to `0` a combined server and client will be created.
		If greater than `0`, the system will create a dedicated server with `n` clients.
		`client_nodes` can be updated, if the value changes and the configuration is applied again, it will attempt to nondestructively
		scale the cluster.

		```hcl
		client_nodes = 3
		```
	*/
	ClientNodes int `hcl:"client_nodes,optional" json:"client_nodes,omitempty"`
	/*
	   An environment map allows you to set environment variables in the container.

	   ```hcl
	   environment = {
	     something   = "PATH"
	     other = "/usr/local/bin"
	   }
	   ```
	*/
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	/*
		Path to a file containing custom Nomad server config to use when creating the server.
		Note: This is only added to server nodes.

		This file extends the default server configuration and is mounted at the path `/etc/nomad.d/server_user_config.hcl` on server nodes.

		```hcl
		server_config = <<-EOF
		server {
		  enabled = true
		  bootstrap_expect = 1
		}

		client {
		  enabled = true
		  meta {
		    node_type = "server"
		  }
		}

		plugin "raw_exec" {
		  config {
		    enabled = true
		  }
		}
		EOF
		```
	*/
	ServerConfig string `hcl:"server_config,optional" json:"server_config,omitempty"`
	/*
		Path to a file containing custom Nomad client config to use when creating the server.
		Note: This file is added to both server and clients nodes.

		This file extends the default client config and is mounted at the path `/etc/nomad.d/client_user_config.hcl`

		```hcl
		client_config = <<-EOF
		client {
		  enabled = true
		  server_join {
		    retry_join = ["%s"]
		  }
		}

		plugin "raw_exec" {
		  config {
		    enabled = true
		  }
		}
		EOF
		```
	*/
	ClientConfig string `hcl:"client_config,optional" json:"client_config,omitempty"`
	/*
		Path to a file containing custom Consul agent config to use when creating the client.

		```hcl
		consul_config = "./files/consul/config.hcl"
		```
	*/
	ConsulConfig string `hcl:"consul_config,optional" json:"consul_config,omitempty"`
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
	Volumes container.Volumes `hcl:"volume,block" json:"volumes,omitempty"`
	/*
		Should a browser window be automatically opened when this resource is created.
		Browser windows will open at the path specified by this property.

		@ignore
	*/
	OpenInBrowser bool `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"`
	/*
		Nomad datacenter for the clients, defaults to `dc1`

		```hcl
		datacenter = "east"
		```
	*/
	Datacenter string `hcl:"datacenter,optional" json:"datacenter"`
	/*
		Docker image in the local Docker image cache to copy to the cluster on creation.
		This image is added to the Nomad clients docker cache enabling jobs to use images that may not be in the local registry.

		Changes to copied images are automatically tracked.
		Should the image change running jumppad up would push any changes to the cluster automatically.

		```hcl
		copy_image {
		  name = "mylocalimage:versoin"
		}
		```

		@type []Image
	*/
	CopyImages container.Images `hcl:"copy_image,block" json:"copy_images,omitempty"`
	/*
		A `port` stanza allows you to expose container ports on the local network or host.
		This stanza can be specified multiple times.

		```
		port {
		  local = 80
		  host  = 8080
		}
		```

		@type []Port
	*/
	Ports container.Ports `hcl:"port,block" json:"ports,omitempty"`
	/*
		A `port_range` stanza allows you to expose a range of container ports on the local network or host.
		This stanza can be specified multiple times.

		The following example would create 11 ports from `80` to `90` (inclusive) and expose them to the host machine.

		```hcl
		port {
		  range       = "80-90"
		  enable_host = true
		}
		```

		@type []PortRange
	*/
	PortRanges container.PortRanges `hcl:"port_range,block" json:"port_ranges,omitempty"`
	/*
		Specifies the configuration for the Nomad cluster.

		```hcl
		config {
		  docker {
		    no_proxy            = ["insecure.container.local.jmpd.in"]
		    insecure_registries = ["insecure.container.local.jmpd.in:5003"]
		  }
		}
		```
	*/
	Config *Config `hcl:"config,block" json:"config,omitempty"`
	/*
		Port to expose the Nomad API on the host.
		By default this uses the standard nomad port 4646; however, if you are running multiple nomad instances you will need
		to override this value.

		```hcl
		api_port = 14646
		```
	*/
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`
	/*
		The port where the Jumppad connector is exposed to the host, this property is requied by the ingress resource and is not
		generally needed when building blueprints.

		@computed
	*/
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`
	/*
		Local directory where the server and client configuration is stored.

		@computed
	*/
	ConfigDir string `hcl:"config_dir,optional" json:"config_dir,omitempty"`
	/*
		The fully qualified resource name for the Nomad server, this value can be used to address the server from the Docker network.
		It is also the name of the Docker container.

		```hcl
		server.name.nomad-cluster.local.jmpd.in
		```

		@computed
	*/
	ServerContainerName string `hcl:"server_container_name,optional" json:"server_container_name,omitempty"`
	/*
		The fully qualified resource names for the Nomad clients, this value can be used to address the client from the Docker network.
		It is also the name of the Docker container.

		When client_nodes is set to `0` this property will have no value.

		```hcl
		[
		  "abse42wsdff.client.name.nomad-cluster.local.jmpd.in",
		  "kjdf23123.client.name.nomad-cluster.local.jmpd.in",
		  "123dfkjs.client.name.nomad-cluster.local.jmpd.in",
		]
		```

		@computed
	*/
	ClientContainerName []string `hcl:"client_container_name,optional" json:"client_container_name,omitempty"`
	/*
		Local IP address of the Nomad server, this property can be used to set the NOAMD_ADDR on the Jumppad client.

		```hcl
		output "NOMAD_ADDR" {
		  value = "http://${resource.nomad_cluster.dev.external_ip}:${resource.nomad_cluster.dev.api_port}"
		}
		```
	*/
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

const nomadBaseImage = "ghcr.io/jumppad-labs/nomad"
const nomadBaseVersion = "v1.8.4"

/*
```hcl

	resource "nomad_cluster" "dev" {
	  config {
	    ...
	  }
	}

```
*/
type Config struct {
	/*
		Specifies configuration for the Docker driver.

		```hcl
		docker {
		  no_proxy            = ["insecure.container.local.jmpd.in"]
		  insecure_registries = ["insecure.container.local.jmpd.in:5003"]
		}
		```
	*/
	DockerConfig *DockerConfig `hcl:"docker,block" json:"docker,omitempty"`
}

/*
```hcl

	resource "nomad_cluster" "dev" {
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
		NoProxy is a list of docker registires that should be excluded from the image cache

		```hcl
		no_proxy = ["insecure.container.local.jmpd.in"]
		```
	*/
	NoProxy []string `hcl:"no_proxy,optional" json:"no-proxy,omitempty"`

	/*
		InsecureRegistries is a list of docker registries that should be treated as insecure

		```hcl
		insecure_registries = ["insecure.container.local.jmpd.in:5003"]
		```
	*/
	InsecureRegistries []string `hcl:"insecure_registries,optional" json:"insecure-registries,omitempty"`
}

func (n *NomadCluster) Process() error {
	if n.Image == nil {
		n.Image = &container.Image{Name: fmt.Sprintf("%s:%s", nomadBaseImage, nomadBaseVersion)}
	}

	if n.ServerConfig != "" {
		n.ServerConfig = utils.EnsureAbsolute(n.ServerConfig, n.Meta.File)
	}

	if n.ClientConfig != "" {
		n.ClientConfig = utils.EnsureAbsolute(n.ClientConfig, n.Meta.File)
	}

	if n.ConsulConfig != "" {
		n.ConsulConfig = utils.EnsureAbsolute(n.ConsulConfig, n.Meta.File)
	}

	if n.Datacenter == "" {
		n.Datacenter = "dc1"
	}

	// Process volumes
	// make sure mount paths are absolute
	for i, v := range n.Volumes {
		if v.Type == "" || v.Type == "bind" {
			// only change path for bind mounts
			n.Volumes[i].Source = utils.EnsureAbsolute(v.Source, n.Meta.File)
		}
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(n.Meta.ID)
		if r != nil {
			state := r.(*NomadCluster)
			n.ExternalIP = state.ExternalIP
			n.ConfigDir = state.ConfigDir
			n.ServerContainerName = state.ServerContainerName
			n.ClientContainerName = state.ClientContainerName
			n.APIPort = state.APIPort
			n.ConnectorPort = state.ConnectorPort

			// add the image ids from the state, this allows the tracking of
			// pushed images so that they can be automatically updated

			// add the image id from state
			for x, img := range n.CopyImages {
				for _, sImg := range state.CopyImages {
					if img.Name == sImg.Name && img.Username == sImg.Username {
						n.CopyImages[x].ID = sImg.ID
					}
				}
			}

			// the network name is set
			copy(n.Networks, state.Networks)
		}
	}

	// set the default port if not set
	if n.APIPort == 0 {
		n.APIPort = 4646
	}

	return nil
}
