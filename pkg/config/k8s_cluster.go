package config

// TypeK8sCluster is the resource string for a Cluster resource
const TypeK8sCluster ResourceType = "k8s_cluster"

// K8sCluster is a config stanza which defines a Kubernetes or a Nomad cluster
type K8sCluster struct {
	// embedded type holding name, etc.
	ResourceInfo

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Driver      string  `hcl:"driver" json:"driver,omitempty"`
	Version     string  `hcl:"version,optional json:"version,omitempty"`
	Nodes       int     `hcl:"nodes,optional json:"nodes,omitempty"`
	Config      []KV    `hcl:"config,block" json:"config,omitempty"`
	Environment []KV    `hcl:"env,block" json:"environment,omitempty"`
	Images      []Image `hcl:"image,block" json:"images,omitempty"`
}

// NewK8sCluster creates new Cluster config with the correct defaults
func NewK8sCluster(name string) *K8sCluster {
	return &K8sCluster{ResourceInfo: ResourceInfo{Name: name, Type: TypeK8sCluster, Status: PendingCreation}}
}
