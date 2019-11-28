package config

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type Cluster struct {
	name       string
	Driver     string `hcl:"driver"`
	Version    string `hcl:"version,optional"`
	Nodes      int    `hcl:"nodes,optional"`
	Network    string `hcl:"network"`
	networkRef *Network
}
