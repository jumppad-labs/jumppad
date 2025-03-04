package k8s

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeK8sConfig defines the string type for the Kubernetes config resource
const TypeK8sConfig string = "k8s_config"
const TypeKubernetesConfig string = "kubernetes_config"

/*
The `kubernetes_config` resource allows Kubernetes configuraton to be applied to a `kubernetes_cluster`.

You can specify a list of paths or individual files and health checks for the resources.
A `kubernetes_config` only completes once the configuration has been successfully applied and any health checks have passed.
This allows you to create complex dependencies for your applications.

The system monitors changes to the config defined in the paths property and automatically recreates this resource when the
configuration is applied.

@resource
*/
type Config struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		The reference to a cluster to apply the jobs to.
		Kubernetes config is only applied when the referenced cluster is created and healthy.

		@example
		```
		resource "kubernetes_config" "example" {
			cluster = resource.kubernetes_cluster.dev
			...
		}
		```
	*/
	Cluster Cluster `hcl:"cluster" json:"cluster"`
	// Paths to the Kubernetes config files to apply to the cluster.
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	/*
		Determines if the resource waits until all config defined in the paths has been accepted and started by the server.
		If set to `false` the resource returns immediately after submitting the job.
	*/
	WaitUntilReady bool `hcl:"wait_until_ready" json:"wait_until_ready"`
	/*
		Optional health check to perform after the jobs have been applied, this resource will not complete until the health
		checks are passing.
	*/
	HealthCheck *healthcheck.HealthCheckKubernetes `hcl:"health_check,block" json:"health_check,omitempty"`
	/*
		JobChecksums store a checksum of the files or paths referenced in the Paths field.
		This is used to detect when a file changes so that it can be re-applied.

		@ignore
	*/
	JobChecksums map[string]string `hcl:"job_checksums,optional" json:"job_checksums,omitempty"`
}

func (k *Config) Process() error {
	// make all the paths absolute
	for i, p := range k.Paths {
		k.Paths[i] = utils.EnsureAbsolute(p, k.Meta.File)
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(k.Meta.ID)
		if r != nil {
			state := r.(*Config)
			k.JobChecksums = state.JobChecksums
		}
	}

	return nil
}
