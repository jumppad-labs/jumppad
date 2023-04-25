package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster string = "nomad_cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	// Port is optional, by default the server exposes port 4646
	Port int `hcl:"port,optional" json:"port,omitempty"`

	Version       string            `hcl:"version,optional" json:"version,omitempty"`
	ClientNodes   int               `hcl:"client_nodes,optional" json:"client_nodes,omitempty"`
	Environment   map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	Images        []Image           `hcl:"image,block" json:"images,omitempty"`
	ServerConfig  string            `hcl:"server_config,optional" json:"server_config,omitempty"`
	ClientConfig  string            `hcl:"client_config,optional" json:"client_config,omitempty"`
	ConsulConfig  string            `hcl:"consul_config,optional" json:"consul_config,omitempty"`
	Volumes       []Volume          `hcl:"volume,block" json:"volumes,omitempty"`                     // volumes to attach to the cluster
	OpenInBrowser bool              `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"` // open the UI in the browser after creation

	// Output Parameters

	// The APIPort the server is running on
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`

	// The Port where the connector is running
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`

	// The directory where the server and client config is written to
	ConfigDir string `hcl:"config_dir,optional" json:"config_dir,omitempty"`

	// The fully qualified docker address for the server
	ServerFQDN string `hcl:"server_fqdn,optional" json:"server_fqdn,omitempty"`

	// The fully qualified docker address for the client nodes
	ClientFQDN []string `hcl:"client_fqdn,optional" json:"client_fqdn,omitempty"`

	// ExternalIP is the ip address of the cluster, this generally resolves
	// to the docker ip
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
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

	// set the default port
	if n.Port == 0 {
		n.Port = 4646
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(n.ID)
		if r != nil {
			kstate := r.(*NomadCluster)
			n.ExternalIP = kstate.ExternalIP
			n.ConfigDir = kstate.ConfigDir
			n.ServerFQDN = kstate.ServerFQDN
			n.ClientFQDN = kstate.ClientFQDN
			n.APIPort = kstate.APIPort
			n.ConnectorPort = kstate.ConnectorPort
		}
	}

	return nil
}
