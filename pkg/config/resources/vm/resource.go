package vm

import (
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeVM is the resource for generating random numbers
const TypeVM string = "vm"

// Allows the creation of virtual machines using libvirt and qemu
type VM struct {
	types.ResourceMetadata `hcl:",remain"`

	Config Config `hcl:"config,block" json:"config"`

	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"`

	Disks []Disk `hcl:"disk,block" json:"disks,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`
	Ports    []Port              `hcl:"port,block" json:"ports,omitempty"`
	Volumes  []Volume            `hcl:"volume,block" json:"volumes,omitempty"`

	VNC VNC `hcl:"vnc,block" json:"vnc,omitempty"`

	CloudInit   *CloudInit `hcl:"cloud_init,block" json:"cloud_init,omitempty"`
	CloudConfig string     `hcl:"cloud_config,optional" json:"cloud_config"`
}

type CloudInit struct {
	NetworkConfig string `hcl:"network_config,optional" json:"network_config"`
	UserData      string `hcl:"user_data,optional" json:"user_data"`
	MetaData      string `hcl:"meta_data,optional" json:"meta_data"`
}

type VNC struct {
	Port int `hcl:"port" json:"port"`
}

type Config struct {
	Arch     string `hcl:"arch,optional" json:"arch"`
	Emulator string `hcl:"emulator,optional" json:"emulator"`
}

type Resources struct {
	CPU    int `hcl:"cpu,optional" json:"cpu,omitempty"`
	Memory int `hcl:"memory,optional" json:"memory,omitempty"`
}

type Disk struct {
	Type   string `hcl:"type" json:"type"` // e.g. ext4
	Source string `hcl:"source,optional" json:"source,omitempty"`
	Size   int    `hcl:"size,optional" json:"size,omitempty"` // size in MB
}

type NetworkAttachment struct {
	ID        string   `hcl:"id" json:"id"`
	IPAddress string   `hcl:"ip_address,optional" json:"ip_address,omitempty"`
	Aliases   []string `hcl:"aliases,optional" json:"aliases,omitempty"`

	// output

	// Name will equal the name of the network as created by jumppad
	Name string `hcl:"name,optional" json:"name,omitempty"`

	// AssignedAddress will equal if IPAddress is set, else it will be the value automatically
	// assigned from the network
	AssignedAddress string `hcl:"assigned_address,optional" json:"assigned_address,omitempty"`
}

type Port struct {
	Local         string `hcl:"local" json:"local"`
	Host          string `hcl:"host,optional" json:"host,omitempty"`
	Protocol      string `hcl:"protocol,optional" json:"protocol,omitempty"`
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser" mapstructure:"open_in_browser"`
}

type Volume struct {
	Source      string `hcl:"source" json:"source"`
	Destination string `hcl:"destination" json:"destination"`
	ReadOnly    bool   `hcl:"read_only,optional" json:"read_only,omitempty"`
}

// func (c *VM) Process() error {
// 	// do we have an existing resource in the state?
// 	// if so we need to set any computed resources for dependents
// 	cfg, err := config.LoadState()
// 	if err == nil {
// 		// try and find the resource in the state
// 		r, _ := cfg.FindResource(c.ID)
// 		if r != nil {
// 			state := r.(*VM)
// 			c.UUID = state.UUID
// 		}
// 	}

// 	return nil
// }
