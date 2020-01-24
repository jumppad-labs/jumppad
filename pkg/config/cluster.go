package config

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type Cluster struct {
	ID         string // unqique id for the resource
	State      State  // current state
	Name       string
	NetworkRef *Network
	WANRef     *Network

	Network string `hcl:"network"`

	Driver      string  `hcl:"driver"`
	Version     string  `hcl:"version,optional"`
	Nodes       int     `hcl:"nodes,optional"`
	Config      []KV    `hcl:"config,block"`
	Environment []KV    `hcl:"env,block"`
	Images      []Image `hcl:"image,block"`
}

// ClusterConfig defines arbitary config to set for the cluster
type ClusterConfig struct {
	ConsulHTTPAddr string `hcl:"consul_http_addr,optional"`
}
