package k8s

import "github.com/jumppad-labs/jumppad/pkg/config"

func init() {
	config.RegisterResource(TypeK8sCluster, &K8sCluster{}, &ClusterProvider{})
	config.RegisterResource(TypeK8sConfig, &K8sConfig{}, &ConfigProvider{})
}
