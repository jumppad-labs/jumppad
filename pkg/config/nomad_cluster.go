package config

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster ResourceType = "cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	ResourceInfo

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Driver      string  `hcl:"driver" json:"driver,omitempty"`
	Version     string  `hcl:"version,optional" json:"version,omitempty"`
	Nodes       int     `hcl:"nodes,optional" json:"nodes,omitempty"`
	Config      []KV    `hcl:"config,block" json:"config,omitempty"`
	Environment []KV    `hcl:"env,block" json:"environment,omitempty"`
	Images      []Image `hcl:"image,block" json:"images,omitempty"`
}

// NewCluster creates new Cluster config with the correct defaults
func NewNomadCluster(name string) *NomadCluster {
	return &NomadCluster{ResourceInfo: ResourceInfo{Name: name, Type: TypeNomadCluster, Status: PendingCreation}}
}

// ClusterConfig defines arbitary config to set for the cluster
type ClusterConfig struct {
	ConsulHTTPAddr string `hcl:"consul_http_addr,optional" json:"consul_http_addr,omitempty"`
}
