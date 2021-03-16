package config

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster ResourceType = "nomad_cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	ResourceInfo `mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Version      string   `hcl:"version,optional" json:"version,omitempty"`
	ClientNodes  int      `hcl:"client_nodes,optional" json:"client_nodes,omitempty" mapstructure:"client_nodes"`
	Nodes        int      `hcl:"nodes,optional" json:"nodes,omitempty"`
	Environment  []KV     `hcl:"env,block" json:"environment,omitempty" mapstructure:"env"`
	Images       []Image  `hcl:"image,block" json:"images,omitempty"`
	ServerConfig string   `hcl:"server_config,optional" json:"server_config,omitempty" mapstructure:"server_config"`
	ClientConfig string   `hcl:"client_config,optional" json:"client_config,omitempty" mapstructure:"client_config"`
	Volumes      []Volume `hcl:"volume,block" json:"volumes,omitempty"` // volumes to attach to the cluster
}

// NewCluster creates new Cluster config with the correct defaults
func NewNomadCluster(name string) *NomadCluster {
	return &NomadCluster{ResourceInfo: ResourceInfo{Name: name, Type: TypeNomadCluster, Status: PendingCreation}}
}

// ClusterConfig defines arbitary config to set for the cluster
type ClusterConfig struct {
	ConsulHTTPAddr string `hcl:"consul_http_addr,optional" json:"consul_http_addr,omitempty" mapstructure:"consul_http_addr"`
}
