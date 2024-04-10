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

// K8sConfig applies and deletes and deletes Kubernetes configuration
type K8sConfig struct {
	types.ResourceBase `hcl:",remain"`

	Cluster K8sCluster `hcl:"cluster" json:"cluster"`

	// Path of a file or directory of Kubernetes config files to apply
	Paths []string `hcl:"paths" validator:"filepath" json:"paths"`
	// WaitUntilReady when set to true waits until all resources have been created and are in a "Running" state
	WaitUntilReady bool `hcl:"wait_until_ready" json:"wait_until_ready"`

	// HealthCheck defines a health check for the resource
	HealthCheck *healthcheck.HealthCheckKubernetes `hcl:"health_check,block" json:"health_check,omitempty"`

	// output

	// JobChecksums stores a checksum of the files or paths
	JobChecksums []string `hcl:"job_checksums,optional" json:"job_checksums,omitempty"`
}

func (k *K8sConfig) Process() error {
	// make all the paths absolute
	for i, p := range k.Paths {
		k.Paths[i] = utils.EnsureAbsolute(p, k.Meta.File)
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(k.Meta.ID)
		if r != nil {
			state := r.(*K8sConfig)
			k.JobChecksums = state.JobChecksums
		}
	}

	return nil
}
