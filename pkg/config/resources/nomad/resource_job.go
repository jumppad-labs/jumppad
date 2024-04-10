package nomad

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeNomadJob defines the string type for the Kubernetes config resource
const TypeNomadJob string = "nomad_job"

// NomadJob applies and deletes and deletes Nomad cluster jobs
type NomadJob struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster NomadCluster `hcl:"cluster" json:"cluster"`

	// Path of a file or directory of Job files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`

	// HealthCheck defines a health check for the resource
	HealthCheck *healthcheck.HealthCheckNomad `hcl:"health_check,block" json:"health_check,omitempty"`

	// output

	// JobChecksums stores a checksum of the files or paths
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
