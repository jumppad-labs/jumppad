package resources

import (
	"github.com/shipyard-run/hclconfig/types"
)

// TypeK8sCluster is the resource string for a Cluster resource
const TypeK8sCluster string = "k8s_cluster"

// K8sCluster is a config stanza which defines a Kubernetes or a Nomad cluster
type K8sCluster struct {
	// embedded type holding name, etc.
	types.ResourceMetadata `hcl:",remain"`

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

	// Path to the Kubernetes config
	KubeConfig string `hcl:"kubeconfig,optional" json:"kubeconfig,omitempty"`

	// Port the API server is running on
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`

	// Port the connector is running on
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`

	// Fully qualified domain name for the container, this address can be
	// used to reference the container within docker and from other containers
	FQDN string `hcl:"fqdn,optional" json:"fqdn,omitempty"`

	// ExternalIP is the ip address of the cluster, this generally resolves
	// to the docker ip
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

func (k *K8sCluster) Process() error {
	for i, v := range k.Volumes {
		k.Volumes[i].Source = ensureAbsolute(v.Source, k.File)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(k.ID)
		if r != nil {
			kstate := r.(*K8sCluster)
			k.KubeConfig = kstate.KubeConfig
			k.FQDN = kstate.FQDN
			k.APIPort = kstate.APIPort
			k.ConnectorPort = kstate.ConnectorPort
			k.ExternalIP = kstate.ExternalIP

			// add the network addresses
			for _, a := range kstate.Networks {
				for i, m := range k.Networks {
					if m.ID == a.ID {
						k.Networks[i].AssignedAddress = a.AssignedAddress
						k.Networks[i].Name = a.Name
						break
					}
				}
			}
		}
	}

	return nil
}
