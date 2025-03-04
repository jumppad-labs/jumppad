package container

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeSidecar is the resource string for a Sidecar resource
const TypeSidecar string = "sidecar"

/*
Sidecar defines a structure for creating Docker containers

@resource
*/
type Sidecar struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	Target Container `hcl:"target" json:"target"`
	// Image defines a Docker image to use when creating the container.
	Image Image `hcl:"image,block" json:"image"`
	// Entrypoint for the container, if not set, Jumppad starts the container using the entrypoint defined in the Docker image.
	Entrypoint []string `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`
	/*
		Command allows you to specify a command to execute when starting a container. Command is specified as an array of strings, each part of the command is a separate string.

		For example, to start a container and follow logs at /dev/null the following command could be used.

		@example
		```
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

		@example
		```
		environment = {
			PATH = "/user/local/bin"
		}
		```
	*/
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	// Labels to apply to the container
	Labels map[string]string `hcl:"labels,optional" json:"labels,omitempty"`
	/*
		A volume allows you to specify a local volume which is mounted to the container when it is created.
		This stanza can be specified multiple times.

		@example
		```
		volume {
			source      = "./"
			destination = "/files"
		}
		```
	*/
	Volumes []Volume `hcl:"volume,block" json:"volumes,omitempty"`
	// Should the container run in Docker privileged mode?
	Privileged bool `hcl:"privileged,optional" json:"privileged,omitempty"`
	// Define resource constraints for the container
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"`
	/*
		Define a health check for the container, the resource will only be marked as successfully created when the health check passes.

		@example
		```
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
	*/
	HealthCheck *healthcheck.HealthCheckContainer `hcl:"health_check,block" json:"health_check,omitempty"`
	// The maximum number of times a container will be restarted when it exits with a status code other than 0
	MaxRestartCount int `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty"`
	/*
		Fully qualified resource name for the container the sidecar is linked to, this can be used to access the sidecar from other sources.

		@example
		```
		name.container.local.jmpd.in
		```

		@computed
	*/
	// ContainerName is the fully qualified domain name for the container the sidecar is linked to, this can be used
	// to access the sidecar from other sources
	ContainerName string `hcl:"container_name,optional" json:"container_name,omitempty"`
}

func (c *Sidecar) Process() error {
	// process volumes
	for i, v := range c.Volumes {
		// make sure mount paths are absolute when type is bind
		if v.Type == "" || v.Type == "bind" {
			c.Volumes[i].Source = utils.EnsureAbsolute(v.Source, c.Meta.File)
		}
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			kstate := r.(*Sidecar)
			c.ContainerName = kstate.ContainerName

			// add the image id from state
			c.Image.ID = kstate.Image.ID
		}
	}

	return nil
}
