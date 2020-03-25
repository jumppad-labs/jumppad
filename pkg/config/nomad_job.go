package config

// TypeNomadJob defines the string type for the Kubernetes config resource
const TypeNomadJob ResourceType = "nomad_config"

// NomadJob applies and deletes and deletes Nomad cluster jobs
type NomadJob struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster" json:"cluster"`
	// Path of a file or directory of Job files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	// WaitUntilReady when set to true waits until all resources have been created and are in a "Running" state
	WaitUntilReady bool `hcl:"wait_until_ready"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block"`
}

// NewNomadJob creates a kubernetes config resource with the correct defaults
func NewNomadJob(name string) *NomadJob {
	return &NomadJob{ResourceInfo: ResourceInfo{Name: name, Type: TypeK8sConfig, Status: PendingCreation}}
}

// Validate the K8sConfig and return errors
func (b *NomadJob) Validate() []error {
	return nil
}
