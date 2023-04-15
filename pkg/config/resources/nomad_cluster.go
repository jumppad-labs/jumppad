package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster string = "nomad_cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Version       string            `hcl:"version,optional" json:"version,omitempty"`
	ClientNodes   int               `hcl:"client_nodes,optional" json:"client_nodes,omitempty"`
	Nodes         int               `hcl:"nodes,optional" json:"nodes,omitempty"`
	Env           map[string]string `hcl:"env,block" json:"env,omitempty"`
	Images        []Image           `hcl:"image,block" json:"images,omitempty"`
	ServerConfig  string            `hcl:"server_config,optional" json:"server_config,omitempty"`
	ClientConfig  string            `hcl:"client_config,optional" json:"client_config,omitempty"`
	ConsulConfig  string            `hcl:"consul_config,optional" json:"consul_config,omitempty"`
	Volumes       []Volume          `hcl:"volume,block" json:"volumes,omitempty"`                     // volumes to attach to the cluster
	OpenInBrowser bool              `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"` // open the UI in the browser after creation

	APIPort       int      `hcl:"api_port,optional" json:"api_port,omitempty"`
	ConnectorPort int      `hcl:"connector_port,optional" json:"connector_port,omitempty"`
	Address       string   `hcl:"server_address,optional" json:"server_address,omitempty"`
	ClientAddress []string `hcl:"client_address,optional" json:"client_address,omitempty"`
	ConfigDir     string   `hcl:"config_dir,optional" json:"config_dir,omitempty"`
}

func (n *NomadCluster) Process() error {
	if n.ServerConfig != "" {
		n.ServerConfig = ensureAbsolute(n.ServerConfig, n.File)
	}

	if n.ClientConfig != "" {
		n.ClientConfig = ensureAbsolute(n.ClientConfig, n.File)
	}

	if n.ConsulConfig != "" {
		n.ConsulConfig = ensureAbsolute(n.ConsulConfig, n.File)
	}

	// Process volumes
	// make sure mount paths are absolute
	for i, v := range n.Volumes {
		n.Volumes[i].Source = ensureAbsolute(v.Source, n.File)
	}

	return nil
}
