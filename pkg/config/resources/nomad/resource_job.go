package nomad

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeNomadJob defines the string type for the Kubernetes config resource
const TypeNomadJob string = "nomad_job"

/*
The `nomad_job` resource allows you to apply one or more Nomad job files to a cluster.

Jumppad monitors changes to the jobs defined in the paths property and automatically recreates this resource when jumppad up is called.

```hcl

	resource "nomad_job" "name" {
	  ...
	}

```

@include healthcheck.HealthCheckNomad

@resource
*/
type NomadJob struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`
	/*
		The reference to a cluster to apply the jobs to.
		Nomad jobs are only applied when the referenced cluster is created and healthy.

		@example
		```
		resource "nomad_job" "example" {
		cluster = resource.nomad_cluster.dev
		...
		}
		```

		@reference nomad.Cluster
	*/
	Cluster NomadCluster `hcl:"cluster" json:"cluster"`
	// Paths to the Nomad job files to apply to the cluster.
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	/*
		Optional health check to perform after the jobs have been applied, this resource will not complete until the health
		checks are passing.
	*/
	HealthCheck *healthcheck.HealthCheckNomad `hcl:"health_check,block" json:"health_check,omitempty"`
	/*
		JobChecksums stores a checksum of the files or paths

		@ignore
	*/
	JobChecksums []string `hcl:"job_checksums,optional" json:"job_checksums,omitempty"`
}

func (n *NomadJob) Process() error {
	// make all the paths absolute
	for i, p := range n.Paths {
		n.Paths[i] = utils.EnsureAbsolute(p, n.Meta.File)
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(n.Meta.ID)
		if r != nil {
			state := r.(*NomadJob)
			n.JobChecksums = state.JobChecksums
		}
	}

	return nil
}
