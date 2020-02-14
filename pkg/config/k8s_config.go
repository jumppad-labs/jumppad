package config

// TypeK8sConfig defines the string type for the Kubernetes config resource
const TypeK8sConfig ResourceType = "k8s_config"

// K8sConfig applies and deletes and deletes Kubernetes configuration
type K8sConfig struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster" json:"cluster"`
	// Path of a file or directory of Kubernetes config files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	// WaitUntilReady when set to true waits until all resources have been created and are in a "Running" state
	WaitUntilReady bool `hcl:"wait_until_ready"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block"`
}

// NewK8sConfig creates a kubernetes config resource with the correct defaults
func NewK8sConfig(name string) *K8sConfig {
	return &K8sConfig{ResourceInfo: ResourceInfo{Name: name, Type: TypeK8sConfig, Status: PendingCreation}}
}

// Validate the K8sConfig and return errors
func (b *K8sConfig) Validate() []error {
	return nil
}
