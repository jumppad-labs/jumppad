package resources

import "github.com/jumppad-labs/hclconfig/types"

// TypeNomadJob defines the string type for the Kubernetes config resource
const TypeNomadJob string = "nomad_job"

// NomadJob applies and deletes and deletes Nomad cluster jobs
type NomadJob struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster" json:"cluster"`

	// Path of a file or directory of Job files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty"`
}

func (n *NomadJob) Process() error {
	// make all the paths absolute
	for i, p := range n.Paths {
		n.Paths[i] = ensureAbsolute(p, n.File)
	}

	return nil
}
