package resources

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster string = "nomad_cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks      []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified
	Image         *Image              `hcl:"image,block" json:"images,omitempty"`     // optional image to use for the cluster
	ClientNodes   int                 `hcl:"client_nodes,optional" json:"client_nodes,omitempty"`
	Environment   map[string]string   `hcl:"environment,optional" json:"environment,omitempty"`
	ServerConfig  string              `hcl:"server_config,optional" json:"server_config,omitempty"`
	ClientConfig  string              `hcl:"client_config,optional" json:"client_config,omitempty"`
	ConsulConfig  string              `hcl:"consul_config,optional" json:"consul_config,omitempty"`
	Volumes       []Volume            `hcl:"volume,block" json:"volumes,omitempty"`                     // volumes to attach to the cluster
	OpenInBrowser bool                `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"` // open the UI in the browser after creation

	// Images that will be copied from the local docker cache to the cluster
	CopyImages []Image `hcl:"copy_image,block" json:"copy_images,omitempty"`

	// Additional ports to expose on the nomad sever node
	Ports      []Port      `hcl:"port,block" json:"ports,omitempty"`             // ports to expose
	PortRanges []PortRange `hcl:"port_range,block" json:"port_ranges,omitempty"` // range of ports to expose

	// Output Parameters

	// The APIPort the server is running on
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`

	// The Port where the connector is running
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`

	// The directory where the server and client config is written to
	ConfigDir string `hcl:"config_dir,optional" json:"config_dir,omitempty"`

	// The fully qualified docker address for the server
	ServerFQRN string `hcl:"server_fqrn,optional" json:"server_fqrn,omitempty"`

	// The fully qualified docker address for the client nodes
	ClientFQRN []string `hcl:"client_fqrn,optional" json:"client_fqrn,omitempty"`

	// ExternalIP is the ip address of the cluster, this generally resolves
	// to the docker ip
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

const nomadBaseImage = "shipyardrun/nomad"
const nomadBaseVersion = "1.6.1"

func (n *NomadCluster) Process() error {
	if n.Image == nil {
		n.Image = &Image{Name: fmt.Sprintf("%s:%s", nomadBaseImage, nomadBaseVersion)}
	}

	if n.ServerConfig != "" {
		n.ServerConfig = utils.EnsureAbsolute(n.ServerConfig, n.File)
	}

	if n.ClientConfig != "" {
		n.ClientConfig = utils.EnsureAbsolute(n.ClientConfig, n.File)
	}

	if n.ConsulConfig != "" {
		n.ConsulConfig = utils.EnsureAbsolute(n.ConsulConfig, n.File)
	}

	// Process volumes
	// make sure mount paths are absolute
	for i, v := range n.Volumes {
		n.Volumes[i].Source = utils.EnsureAbsolute(v.Source, n.File)
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
			n.ServerFQRN = kstate.ServerFQRN
			n.ClientFQRN = kstate.ClientFQRN
			n.APIPort = kstate.APIPort
			n.ConnectorPort = kstate.ConnectorPort

			// add the image ids from the state, this allows the tracking of
			// pushed images so that they can be automatically updated

			// add the image id from state
			for x, img := range n.CopyImages {
				for _, sImg := range kstate.CopyImages {
					if img.Name == sImg.Name && img.Username == sImg.Username {
						n.CopyImages[x].ID = sImg.ID
					}
				}
			}
		}
	}

	// set the default port if not set
	if n.APIPort == 0 {
		n.APIPort = 4646
	}

	return nil
}
