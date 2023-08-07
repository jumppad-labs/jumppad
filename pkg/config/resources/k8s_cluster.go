package resources

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
)

// TypeK8sCluster is the resource string for a Cluster resource
const TypeK8sCluster string = "k8s_cluster"

// K8sCluster is a config stanza which defines a Kubernetes or a Nomad cluster
type K8sCluster struct {
	// embedded type holding name, etc.
	types.ResourceMetadata `hcl:",remain"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image   *Image   `hcl:"image,block" json:"images,omitempty"` // optional image to use when creating the cluster
	Nodes   int      `hcl:"nodes,optional" json:"nodes,omitempty"`
	Volumes []Volume `hcl:"volume,block" json:"volumes,omitempty"` // volumes to attach to the cluster

	// Images that will be copied from the local docker cache to the cluster
	CopyImages []Image `hcl:"copy_image,block" json:"copy_images,omitempty"`

	Ports      []Port      `hcl:"port,block" json:"ports,omitempty"`             // ports to expose
	PortRanges []PortRange `hcl:"port_range,block" json:"port_ranges,omitempty"` // range of ports to expose

	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"` // environment variables to set when starting the container

	// output parameters

	// Path to the Kubernetes config
	KubeConfig string `hcl:"kubeconfig,optional" json:"kubeconfig,omitempty"`

	// Port the API server is running on
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`

	// Port the connector is running on
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`

	// Fully qualified domain name for the container, this address can be
	// used to reference the container within docker and from other containers
	FQRN string `hcl:"fqrn,optional" json:"fqrn,omitempty"`

	// ExternalIP is the ip address of the cluster, this generally resolves
	// to the docker ip
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

const k3sBaseImage = "shipyardrun/k3s"
const k3sBaseVersion = "v1.27.4"

func (k *K8sCluster) Process() error {
	if k.APIPort == 0 {
		k.APIPort = 443
	}

	if k.Image == nil {
		k.Image = &Image{Name: fmt.Sprintf("%s:%s", k3sBaseImage, k3sBaseVersion)}
	}

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
			k.FQRN = kstate.FQRN
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

			// add the image id from state
			for x, img := range k.CopyImages {
				for _, sImg := range kstate.CopyImages {
					if img.Name == sImg.Name && img.Username == sImg.Username {
						k.CopyImages[x].ID = sImg.ID
					}
				}
			}
		}
	}

	return nil
}
