package config

// K8sConfig applies and deletes and deletes Kubernetes configuration
type K8sConfig struct {
	Name       string
	State      State
	ClusterRef *Cluster

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster"`
	// Path of a file or directory of Kubernetes config files to apply
	Paths []string `hcl:"paths" validator:"filepath"`
	// WaitUntilReady when set to true waits until all resources have been created and are in a "Running" state
	WaitUntilReady bool `hcl:"wait_until_ready"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block"`
}

// Validate the K8sConfig and return errors
func (b *K8sConfig) Validate() []error {
	return nil
}
