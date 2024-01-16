package nomad

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCluster is the resource string for a Cluster resource
const TypeNomadCluster string = "nomad_cluster"

// Cluster is a config stanza which defines a Kubernetes or a Nomad cluster
type NomadCluster struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks      ctypes.NetworkAttachments `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified
	Image         *ctypes.Image             `hcl:"image,block" json:"images,omitempty"`     // optional image to use for the cluster
	ClientNodes   int                       `hcl:"client_nodes,optional" json:"client_nodes,omitempty"`
	Environment   map[string]string         `hcl:"environment,optional" json:"environment,omitempty"`
	ServerConfig  string                    `hcl:"server_config,optional" json:"server_config,omitempty"`
	ClientConfig  string                    `hcl:"client_config,optional" json:"client_config,omitempty"`
	ConsulConfig  string                    `hcl:"consul_config,optional" json:"consul_config,omitempty"`
	Volumes       ctypes.Volumes            `hcl:"volume,block" json:"volumes,omitempty"`                     // volumes to attach to the cluster
	OpenInBrowser bool                      `hcl:"open_in_browser,optional" json:"open_in_browser,omitempty"` // open the UI in the browser after creation

	Datacenter string `hcl:"datacenter,optional" json:"datacenter"` // Nomad datacenter, defaults dc1

	// Images that will be copied from the local docker cache to the cluster
	CopyImages ctypes.Images `hcl:"copy_image,block" json:"copy_images,omitempty"`

	// Additional ports to expose on the nomad sever node
	Ports      ctypes.Ports      `hcl:"port,block" json:"ports,omitempty"`             // ports to expose
	PortRanges ctypes.PortRanges `hcl:"port_range,block" json:"port_ranges,omitempty"` // range of ports to expose

	// Configuration for the drivers
	Config *Config `hcl:"config,block" json:"config,omitempty"`

	// Output Parameters

	// The APIPort the server is running on
	APIPort int `hcl:"api_port,optional" json:"api_port,omitempty"`

	// The Port where the connector is running
	ConnectorPort int `hcl:"connector_port,optional" json:"connector_port,omitempty"`

	// The directory where the server and client config is written to
	ConfigDir string `hcl:"config_dir,optional" json:"config_dir,omitempty"`

	// The fully qualified docker address for the server
	ServerContainerName string `hcl:"server_container_name,optional" json:"server_container_name,omitempty"`

	// The fully qualified docker address for the client nodes
	ClientContainerName []string `hcl:"client_container_name,optional" json:"client_container_name,omitempty"`

	// ExternalIP is the ip address of the cluster, this generally resolves
	// to the docker ip
	ExternalIP string `hcl:"external_ip,optional" json:"external_ip,omitempty"`
}

const nomadBaseImage = "shipyardrun/nomad"
const nomadBaseVersion = "1.6.1"

type Config struct {
	// Specifies configuration for the Docker driver.
	DockerConfig *DockerConfig `hcl:"docker,block" json:"docker,omitempty"`
}

type DockerConfig struct {
	// NoProxy is a list of docker registires that should be excluded from the image cache
	NoProxy []string `hcl:"no_proxy,optional" json:"no-proxy,omitempty"`

	// InsecureRegistries is a list of docker registries that should be treated as insecure
	InsecureRegistries []string `hcl:"insecure_registries,optional" json:"insecure-registries,omitempty"`
}

func (n *NomadCluster) Process() error {
	if n.Image == nil {
		n.Image = &ctypes.Image{Name: fmt.Sprintf("%s:%s", nomadBaseImage, nomadBaseVersion)}
	}

	if n.ServerConfig != "" {
		n.ServerConfig = utils.EnsureAbsolute(n.ServerConfig, n.ResourceFile)
	}

	if n.ClientConfig != "" {
		n.ClientConfig = utils.EnsureAbsolute(n.ClientConfig, n.ResourceFile)
	}

	if n.ConsulConfig != "" {
		n.ConsulConfig = utils.EnsureAbsolute(n.ConsulConfig, n.ResourceFile)
	}

	if n.Datacenter == "" {
		n.Datacenter = "dc1"
	}

	// Process volumes
	// make sure mount paths are absolute
	for i, v := range n.Volumes {
		if v.Type == "" || v.Type == "bind" {
			// only change path for bind mounts
			n.Volumes[i].Source = utils.EnsureAbsolute(v.Source, n.ResourceFile)
		}
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	c, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := c.FindResource(n.ResourceID)
		if r != nil {
			state := r.(*NomadCluster)
			n.ExternalIP = state.ExternalIP
			n.ConfigDir = state.ConfigDir
			n.ServerContainerName = state.ServerContainerName
			n.ClientContainerName = state.ClientContainerName
			n.APIPort = state.APIPort
			n.ConnectorPort = state.ConnectorPort

			// add the image ids from the state, this allows the tracking of
			// pushed images so that they can be automatically updated

			// add the image id from state
			for x, img := range n.CopyImages {
				for _, sImg := range state.CopyImages {
					if img.Name == sImg.Name && img.Username == sImg.Username {
						n.CopyImages[x].ID = sImg.ID
					}
				}
			}

			// the network name is set
			for x, net := range state.Networks {
				n.Networks[x] = net
			}
		}
	}

	// set the default port if not set
	if n.APIPort == 0 {
		n.APIPort = 4646
	}

	return nil
}
