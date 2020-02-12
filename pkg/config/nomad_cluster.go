package config

// TypeCluster is the resource string for a Cluster resource
const TypeCluster ResourceType = "cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type Cluster struct {
	// embedded type holding name, etc
	ResourceInfo

	// Network is the reference to a Network resrouce
	Network string `hcl:"network" json:"network,omitempty"`

	Driver      string  `hcl:"driver" json:"driver,omitempty"`
	Version     string  `hcl:"version,optional json:"version,omitempty"`
	Nodes       int     `hcl:"nodes,optional json:"nodes,omitempty"`
	Config      []KV    `hcl:"config,block" json:"config,omitempty"`
	Environment []KV    `hcl:"env,block" json:"environment,omitempty"`
	Images      []Image `hcl:"image,block" json:"images,omitempty"`
}

// NewCluster creates new Cluster config with the correct defaults
func NewCluster(name string) *Cluster {
	return &Cluster{ResourceInfo: ResourceInfo{Name: name, Type: TypeCluster, Status: PendingCreation}}
}

// ClusterConfig defines arbitary config to set for the cluster
type ClusterConfig struct {
	ConsulHTTPAddr string `hcl:"consul_http_addr,optional" json:"consul_http_addr,omitempty"`
}
