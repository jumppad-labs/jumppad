package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeIngress is the resource string for the type
const TypeIngress string = "ingress"

const (
	IngressSourceLocal  = "local"
	IngressSourceK8s    = "k8s"
	IngressSourceDocker = "docker"
)

// Ingress defines an ingress service mapping ports between local host and resources like containers and kube cluster
type Ingress struct {
	types.ResourceMetadata `hcl:",remain"`

	// Id stores the ID of the created connector service
	Id string `json:"id" state:"true"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Destination Traffic `hcl:"destination,block" json:"destination"`
	Source      Traffic `hcl:"source,block" json:"source"`
}

// Traffic defines either a source or a destination block for ingress traffic
type Traffic struct {
	// Driver to use when creating the ingress, k8s, nomad, docker, local
	Driver string `hcl:"driver" json:"driver"`

	// Config is an collection which has driver specific content
	Config TrafficConfig `hcl:"config,block" json:"config"`
}

// TrafficConfig defines the parameters for the traffic
type TrafficConfig struct {
	Cluster       string `hcl:"cluster,optional" json:"cluster,omitempty"`
	Address       string `hcl:"address,optional" json:"address,omitempty"`
	Port          string `hcl:"port" json:"port"`
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"`
}
