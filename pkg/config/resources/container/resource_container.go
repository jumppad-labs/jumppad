package container

import (
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeContainer is the resource string for a Container resource
const TypeContainer string = "container"

/*
Container defines a structure for creating Docker containers

```hcl

	resource "container" "name" {
	  ...
	}

```

@include container.NetworkAttachment
@include container.Image
@include container.Volume
@include container.Port
@include container.PortRange
@include container.Resources
@include container.Capabilities
@include container.User
@include healthcheck.HealthCheckContainer
@include healthcheck.HealthCheckHTTP
@include healthcheck.HealthCheckTCP
@include healthcheck.HealthCheckExec

@resource

@example Minimal Example
```

	resource "container" "unique_name" {
	    network {
	        id         = resource.network.cloud.meta.id
	        ip_address = "10.16.0.203"
	        aliases    = ["my_unique_name_ip_address"]
	    }

	    image {
	        name = "consul:1.6.1"
	    }
	}

```

@example Full Example
```

	resource "container" "unique_name" {
	    depends_on = ["resource.container.another"]

	    network {
	        id         = resource.network.cloud.meta.id
	        ip_address = "10.16.0.200"
	        aliases    = ["my_unique_name_ip_address"]
	    }

	    image {
	        name     = "consul:1.6.1"
	        username = "repo_username"
	        password = "repo_password"
	    }

	    command = [
	        "consul",
	        "agent"
	    ]

	    environment = {
	        CONSUL_HTTP_ADDR = "http://localhost:8500"
	    }

	    volume {
	        source      = "./config"
	        destination = "/config"
	    }

	    port {
	        local  = 8500
	        remote = 8500
	        host   = 18500
	    }

	    port_range {
	        range       = "9000-9002"
	        enable_host = true
	    }

	    privileged = false
	}

```
*/
type Container struct {
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
	*/
	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`
	/*
		Image defines a Docker image to use when creating the container.

		```hcl
		image {
		  name = "redis:latest"
		}
		```
	*/
	Image Image `hcl:"image,block" json:"image"`
	// Entrypoint for the container, if not set, Jumppad starts the container using the entrypoint defined in the Docker image.
	Entrypoint []string `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`
	/*
		Command allows you to specify a command to execute when starting a container. Command is specified as an array of strings, each part of the command is a separate string.

		For example, to start a container and follow logs at /dev/null the following command could be used.

		```hcl
		command = [
		  "tail",
		  "-f",
		  "/dev/null"
		]
		```
	*/
	Command []string `hcl:"command,optional" json:"command,omitempty"`
	/*
		Allows you to set environment variables in the container.

		```hcl
		environment = {
		  PATH = "/user/local/bin"
		}
		```
	*/
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	/*
		Labels to apply to the container

		```hcl
		labels = {
		  key = "value"
		}
		```
	*/
	Labels map[string]string `hcl:"labels,optional" json:"labels,omitempty"`
	/*
		A volume allows you to specify a local volume which is mounted to the container when it is created.
		This stanza can be specified multiple times.

		```hcl
		volume {
		  source      = "./"
		  destination = "/files"
		}
		```
	*/
	Volumes []Volume `hcl:"volume,block" json:"volumes,omitempty"`
	/*
		A port stanza allows you to expose container ports on the local network or host.
		This stanza can be specified multiple times.

		```hcl
		port {
		  local = 80
		  host  = 8080
		}
		```
	*/
	Ports []Port `hcl:"port,block" json:"ports,omitempty"`
	/*
		A port_range stanza allows you to expose a range of container ports on the local network or host.
		This stanza can be specified multiple times.

		The following example would create 11 ports from 80 to 90 (inclusive) and expose them to the host machine.

		```hcl
		port_range {
		  range       = "80-90"
		  enable_host = true
		}
		```
	*/
	PortRanges []PortRange `hcl:"port_range,block" json:"port_ranges,omitempty"`
	/*
		The nameservers to use to resolve dns requests.

		```hcl
		dns = ["1.1.1.1", "8.8.8.8"]
		```
	*/
	DNS []string `hcl:"dns,optional" json:"dns,omitempty"`
	/*
		Should the container run in Docker privileged mode?

		```hcl
		privileged = true
		```
	*/
	Privileged bool `hcl:"privileged,optional" json:"privileged,omitempty"`
	/*
		The capabilities to add or drop.

		```hcl
		capabilities {
		  add = ["AUDIT_CONTROL"]
			drop = ["SETUID"]
		}
		```
	*/
	Capabilities *Capabilities `hcl:"capabilities,block" json:"capabilities,omitempty"`
	/*
		The maximum number of times a container will be restarted when it exits with a status code other than 0

		```hcl
		max_restart_count = 3
		```
	*/
	MaxRestartCount int `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty"`
	/*
		Define resource constraints for the container

		```hcl
		resources {
		  cpu = 100
			memory = 1024
		}
		```
	*/
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"`
	/*
		Define a health check for the container, the resource will only be marked as successfully created when the health check passes.

		```hcl
		health_check {
		  timeout = "30s"
		  http {
		    address = "http://localhost:8500/v1/status/leader"
		    success_codes = [200]
		  }

		  tcp {
		    address = "localhost:8500"
		  }

		  exec {
		    script = <<-EOF
		      #!/bin/bash

		      curl "http://localhost:9090"
		    EOF
		  }
		}
		```

		@type HealthCheckContainer
	*/
	HealthCheck *healthcheck.HealthCheckContainer `hcl:"health_check,block" json:"health_check,omitempty"`
	/*
		Allows the container to be run as a specific user or group.

		```hcl
		run_as = "ubuntu"
		```
	*/
	RunAs *User `hcl:"run_as,block" json:"run_as,omitempty"`
	/*
		Fully qualified resource name for the container, this value can be used to access the container from within the Docker network.
		`container_name` is also the name of the created Docker container.

		```hcl
		name.container.local.jmpd.in
		```

		@computed
	*/
	ContainerName string `hcl:"container_name,optional" json:"container_name,omitempty"`
}

/*
User and Group configuration to be used when running a container, by default Docker runs commands in the container as root id 0.

```hcl

	user {
	  ...
	}

```
*/
type User struct {
	/*
		Linux user ID or user name to run the container as, this overrides the default user configured in the container image.

		```hcl
		user = "root"
		```
	*/
	User string `hcl:"user" json:"user,omitempty"`
	/*
		Linux group ID or group name to run the container as, this overrides the default group configured in the container image.

		group = "root"
	*/
	Group string `hcl:"group" json:"group,omitempty"`
}

/*
Network attachment defines a network to which the container is attached.

```hcl

	network {
	  ...
	}

```
*/
type NetworkAttachment struct {
	/*
		ID of the network to attach the container to, specified in reference format. e.g. to attach to a network called `cloud`.

		```hcl
		id = resource.network.main.meta.id
		```
	*/
	ID string `hcl:"id" json:"id"`
	/*
		Static IP address to assign container for the network, the ip address must be within range defined by the network subnet.
		If this parameter is omitted an IP address will be automatically assigned.

		```hcl
		ip_address = "10.0.5.24"
		```
	*/
	IPAddress string `hcl:"ip_address,optional" json:"ip_address,omitempty"`
	/*
		Aliases allow alternate names to specified for the container.
		Aliases can be used to reference a container across the network, the container will respond to ping and other network
		resolution using the primary assigned name `[name].container.shipyard.run` and the aliases.

		```hcl
		aliases = [
		  "alt1.container.local.jmpd.in",
		  "alt2.container.local.jmpd.in"
		]
		```
	*/
	Aliases []string `hcl:"aliases,optional" json:"aliases,omitempty"`
	/*
		Name will equal the name of the network as created by jumppad.

		@computed
	*/
	Name string `hcl:"name,optional" json:"name,omitempty"`
	/*
		`assigned_address` will equal the assigned IP address for the network.
		This will equal ip_address if set; otherwise, this is the automatically assigned IP address.

		@computed
	*/
	AssignedAddress string `hcl:"assigned_address,optional" json:"assigned_address,omitempty"`
}

type NetworkAttachments []NetworkAttachment

/*
A resources type allows you to configure the maximum resources which can be consumed.

```hcl

	container {
	  resources {
	    ...
	  }
	}

```
*/
type Resources struct {
	/*
		Set the maximum CPU which can be consumed by the container in MHz, 1 CPU == 1000MHz.

		```hcl
		cpu = 100
		```
	*/
	CPU int `hcl:"cpu,optional" json:"cpu,omitempty"`
	/*
			Pin the container CPU consumption to one or more logical CPUs. For example to pin the container to the core 1 and 4.

		```hcl
		cpi_pin = [1,4]
		```
	*/
	CPUPin []int `hcl:"cpu_pin,optional" json:"cpu_pin,omitempty"`
	/*
		Maximum memory which a container can consume, specified in Megabytes.

		```hcl
		memory = 2048
		```
	*/
	Memory int `hcl:"memory,optional" json:"memory,omitempty"`
	/*
		GPU settings to pass through to container

		```hcl
		gpu {
		  driver = "nvidia"
		  device_ids = ["0", "1"]
		}
		```
	*/
	GPU *GPU `hcl:"gpu,block" json:"gpu,omitempty"`
}

/*
GPU support allows you to pass through GPU devices to the container, this is useful for running GPU accelerated workloads.

For more information on GPU support in Docker see the [official documentation](https://docs.docker.com/desktop/gpu/).

```hcl

	container {
	  resources {
	    gpu {
	      ...
	    }
	  }
	}

```
*/
type GPU struct {
	/*
		The GPU driver to use, i.e "nvidia", note: This has not been tested this with AMD or other GPUs.

		```hcl
		driver = "nvidia"
		```
	*/
	Driver string `hcl:"driver" json:"driver"`
	/*
		The GPUs to pass to the container, i.e "0", "1", "2".

		```hcl
		device_ids = ["0", "1"]
		```
	*/
	DeviceIDs []string `hcl:"device_ids" json:"device_ids"`
}

/*
```hcl

	container {
	  capabilities {
	    ...
	  }
	}

```
*/
type Capabilities struct {
	/*
		CapAdd is a list of kernel capabilities to add to the container

		```hcl
		add = ["IPC_LOCK"]
		```
	*/
	Add []string `hcl:"add,optional" json:"add"`
	/*
		CapDrop is a list of kernel capabilities to remove from the container

		```hcl
		drop = ["SETUID"]
		```
	*/
	Drop []string `hcl:"drop,optional" json:"drop"`
}

/*
A volume type allows the specification of an attached volume.

```hcl

	container {
	  volume {
	    ...
	  }
	}

```
*/
type Volume struct {
	/*
		The source volume to mount in the container, can be specified as a relative `./` or absolute path `/usr/local/bin`.
		Relative paths are relative to the file declaring the container.

		```hcl
		source = "./files/nginx"
		```
	*/
	Source string `hcl:"source" json:"source"`
	/*
		The destination in the container to mount the volume to, must be an absolute path.

		```hcl
		destination = "/etc/nginx"
		```
	*/
	Destination string `hcl:"destination" json:"destination"`
	/*
		The type of the mount, can be one of the following values:

		- bind: bind the source path to the destination path in the container
		- volume: source is a Docker volume
		- tmpfs: create a temporary filesystem

		```hcl
		type = "bind"
		```
	*/
	Type string `hcl:"type,optional" json:"type,omitempty"`
	/*
		Whether or not the volume is read only.

		```hcl
		read_only = true
		```
	*/
	ReadOnly bool `hcl:"read_only,optional" json:"read_only,omitempty"`
	/*
		Configures bind propagation for Docker volume mounts, only applies to bind mounts, can be one of the following values:

		- shared
		- slave
		- private
		- rslave
		- rprivate

		For more information please see the Docker documentation https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation

		```hcl
		bind_propagation = "shared"
		```
	*/
	BindPropagation string `hcl:"bind_propagation,optional" json:"bind_propagation,omitempty"`
	/*
		Configures recursiveness of the bind mount.

		By default Docker mounts with the equivalent of `mount --rbind` meaning that mounts below the the source directory are visible in the container.
		or instance running `docker run --rm --mount type=bind,src=/,target=/host,readonly` busybox will make `/run` of the host available as
		`/host/run` in the container. To make matters even worse it will be writable (since only the toplevel bind is set readonly, not the children).

		If `bind_propagation_non_recursive` is set to true then the container will only see an empty `/host/run`, meaning the
		`tmpfs` which is typically mounted to `/run` on the host is not propagated into the container.

		```hcl
		bind_propagation_non_recursive = true
		```
	*/
	BindPropagationNonRecursive bool `hcl:"bind_propagation_non_recursive,optional" json:"bind_propagation_non_recursive,omitempty"`
	/*
		Configures Selinux relabeling for the container (usually specified as :z or :Z) and can be one of the following values:

		- shared (Equivalent to :z)
		- private (Equivalent to :Z)

		```hcl
		selinux_relabel = "shared"
		```
	*/
	SelinuxRelabel string `hcl:"selinux_relabel,optional" json:"selinux_relabel,omitempty"`
}

type Volumes []Volume

func (c *Container) Process() error {
	// process volumes
	for i, v := range c.Volumes {
		// make sure mount paths are absolute when type is bind, unless this is the docker sock
		if v.Type == "" || v.Type == "bind" {
			c.Volumes[i].Source = utils.EnsureAbsolute(v.Source, c.Meta.File)
		}
	}

	// make sure line endings are linux
	if c.HealthCheck != nil {
		for i := range c.HealthCheck.Exec {
			c.HealthCheck.Exec[i].Script = strings.Replace(c.HealthCheck.Exec[i].Script, "\r\n", "\n", -1)
		}
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			kstate := r.(*Container)
			c.ContainerName = kstate.ContainerName

			// add the image id from state
			c.Image.ID = kstate.Image.ID

			// add the network addresses
			for _, a := range kstate.Networks {
				for i, m := range c.Networks {
					if m.ID == a.ID {
						c.Networks[i].AssignedAddress = a.AssignedAddress
						c.Networks[i].Name = a.Name
						break
					}
				}
			}
		}
	}

	return nil
}
