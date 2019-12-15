package config

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type Cluster struct {
	Name       string
	Driver     string `hcl:"driver"`
	Version    string `hcl:"version,optional"`
	Nodes      int    `hcl:"nodes,optional"`
	Network    string `hcl:"network"`
	NetworkRef *Network
	WANRef     *Network
	Config     []KV    `hcl:"config,block"`
	Images     []Image `hcl:"image,block"`
}

// ClusterConfig defines arbitary config to set for the cluster
type ClusterConfig struct {
	ConsulHTTPAddr string `hcl:"consul_http_addr,optional"`
}

// Image defines a docker image which will be pushed to the clusters Docker
// registry
type Image struct {
	Name string `hcl:"name"`
}
