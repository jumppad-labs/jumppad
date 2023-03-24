package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeK8sCluster is the resource string for a Cluster resource
const TypeK8sCluster string = "k8s_cluster"

// K8sCluster is a config stanza which defines a Kubernetes or a Nomad cluster
type K8sCluster struct {
	// embedded type holding name, etc.
	types.ResourceMetadata `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Driver  string   `hcl:"driver" json:"driver,omitempty"`
	Version string   `hcl:"version,optional" json:"version,omitempty"`
	Nodes   int      `hcl:"nodes,optional" json:"nodes,omitempty"`
	Images  []Image  `hcl:"image,block" json:"images,omitempty"`
	Volumes []Volume `hcl:"volume,block" json:"volumes,omitempty"` // volumes to attach to the cluster

	Ports      []Port      `hcl:"port,block" json:"ports,omitempty"`             // ports to expose
	PortRanges []PortRange `hcl:"port_range,block" json:"port_ranges,omitempty"` // range of ports to expose

	Env map[string]string `hcl:"env_var,optional" json:"env_var,omitempty"` // environment variables to set when starting the container

	// output parameters
	K8sConfig string `hcl:"k8s_config,optional" json:"k8s_config,omitempty"`
}

func (k *K8sCluster) Process() error {
	for i, v := range k.Volumes {
		k.Volumes[i].Source = ensureAbsolute(v.Source, k.File)
	}

	return nil
}
