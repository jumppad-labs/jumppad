package resources

import "github.com/jumppad-labs/hclconfig/types"

// TypeK8sConfig defines the string type for the Kubernetes config resource
const TypeK8sConfig string = "k8s_config"

// K8sConfig applies and deletes and deletes Kubernetes configuration
type K8sConfig struct {
	types.ResourceMetadata `hcl:",remain"`

	// Cluster is the name of the cluster to apply configuration to
	Cluster string `hcl:"cluster" json:"cluster"`
	// Path of a file or directory of Kubernetes config files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	// WaitUntilReady when set to true waits until all resources have been created and are in a "Running" state
	WaitUntilReady bool `hcl:"wait_until_ready" json:"wait_until_ready"`

	// HealthCheck defines a health check for the resource
	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty"`
}

func (k *K8sConfig) Process() error {
	// make all the paths absolute
	for i, p := range k.Paths {
		k.Paths[i] = ensureAbsolute(p, k.File)
	}

	return nil
}
