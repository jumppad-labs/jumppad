package config

// TypeNomadJob defines the string type for the Kubernetes config resource
const TypeNomadJob ResourceType = "nomad_job"

// NomadJob applies and deletes and deletes Nomad cluster jobs
type NomadJob struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster" json:"cluster"`
	// Path of a file or directory of Job files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty" mapstructure:"health_check"`
}

// NewNomadJob creates a kubernetes config resource with the correct defaults
func NewNomadJob(name string) *NomadJob {
	return &NomadJob{ResourceInfo: ResourceInfo{Name: name, Type: TypeNomadJob, Status: PendingCreation}}
}

// Validate the K8sConfig and return errors
func (b *NomadJob) Validate() []error {
	return nil
}
