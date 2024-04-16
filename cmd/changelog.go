package cmd

import (
	"strings"

	"github.com/jumppad-labs/jumppad/cmd/changelog"
	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show the changelog",
	Long:  `Show the changelog`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cl := &changelog.Changelog{}

		// replace """ with ``` in changelog
		changes = strings.ReplaceAll(changes, `"""`, "```")

		err := cl.Show(changes, changesVersion, true)
		if err != nil {
			showErr(err)
		}
	},
}

var changesVersion = "v0.10.5"

var changes = `
## version v0.10.5

## New Features:
Adds outputs for 'exec' resources

Exec resources now have a new parameter 'output' which is a map of key value pairs.
Values for output can be set by echoing a key value to the file '${EXEC_OUTPUT}' in 
the defined script for either remote or local exec.

"""hcl
resource "exec" "install" {
  # Add the output
  echo "exec=install" >> $EXEC_OUTPUT
  echo "foo=bar" >> $EXEC_OUTPUT
  EOF

  timeout = "30s"
}

output "local_exec_install" {
  value = resource.exec.install.output.exec
}
"""

## version v0.10.4
Enable experimental support for nvidia GPUs for container resources

This feature configures the container to use the nvidia runtime and the nvidia
device plugin to access the GPU.  Currently this has only been tested with WSL2 and
Nvidia GPUs.

"""hcl
resource "container" "gpu_test" {
  image {
    name = "nvcr.io/nvidia/k8s/cuda-sample:nbody"
  }

  command = ["nbody", "-gpu", "-benchmark"]

  resources {
    gpu {
      driver     = "nvidia"
      device_ids = ["0"]
    }
  }
}
"""

## version v0.10.1
* Ensure that the total compute is set correctly	for Nomad clusters when 
	running on Docker in Apple Silicon.

## version v0.10.0

## New Features:
* Add experimental cancellation for long running commands, you can
  now press 'ctrl-c' to interupt 'up' and 'down' commands
* Add --force flag to ignore graceful exit for the down command

### Breaking Changes: 

#### Exec Local and Exec Remote Resources
The 'exec_local' and 'exec_remote' resources have been removed in favor
of the new 'exec' resource. The 'exec' resource supports all the functionality
of the old resources and more.

"""hcl
resource "container" "alpine" {
  image {
    name = "alpine"
  }

  command = ["tail", "-f", "/dev/null"]

  volume {
    source      = data("test")
    destination = "/data"
  }
}

resource "exec" "run" {
  script = <<-EOF
  #!/bin/sh
  ${data("test")}/consul agent -dev
  EOF

  daemon = true
}
"""

#### Kubernetes Cluster Configuration

Prior to this version Kubernetes clusters could access the config path like
the following example:

"""
resource "k8s_cluster" "k3s" {
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}
"""

In the latest version this has changed to expand the details of the kubeconfig
providing access to the cluster ca certificate, client certificate and client key.

An updated example can be seen below:

"""
resource "k8s_cluster" "k3s" {
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kube_config.path
}

output "KUBE_CA" {
  value = resource.k8s_cluster.k3s.kube_config.ca
}

output "KUBE_CLIENT_CERT" {
  value = resource.k8s_cluster.k3s.kube_config.client_certificate
}

output "KUBE_CLIENT_KEY" {
  value = resource.k8s_cluster.k3s.kube_config.client_key
}
"""
`
