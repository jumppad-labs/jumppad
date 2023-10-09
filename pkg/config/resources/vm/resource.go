package vm

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

// TypeVM is the resource for generating random numbers
const TypeVM string = "vm"

/*
resource "vm" "test" {
	config {
		arch = "x86_64" // default -> host arch
		emulator = "qemu"
	}

  image = "/path/to/vm-image.qcow2" // .iso .img

  resources {
    cpus = 2
    memory = 4096 // mb
  }

  disk "name" {
    type = "ext4"
    size = 100 // mb
  }

  volume {
    source = "/path/on/host"
    destination = "/path/in/vm"
  }

  network {
    id = resource.network.main.id
    ip_address = "10.0.10.5"
  }

  port {
    local  = 8000
    remote = 8000
    host   = 8000
  }

  cloud_config = <<-EOF
  runcmd: |-
    apt update
    apt install -y curl
  EOF
}
*/

// allows the generation of random numbers
type VM struct {
	types.ResourceMetadata `hcl:",remain"`

	Image string `hcl:"image" json:"image"`

	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"`

	Disks []Disk `hcl:"disk,block" json:"disks,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`
	Ports    []Port              `hcl:"port,block" json:"ports,omitempty"`
	Volumes  []Volume            `hcl:"volume,block" json:"volumes,omitempty"`

	CloudConfig string `hcl:"cloud_config" json:"cloud_config"`
}

type Resources struct {
	CPU    int `hcl:"cpu,optional" json:"cpu,omitempty"`
	Memory int `hcl:"memory,optional" json:"memory,omitempty"`
}

type Disk struct {
	Name string `hcl:"name,label" json:"name"`
	Type string `hcl:"type" json:"type"` // e.g. ext4
	Size int    `hcl:"size" json:"size"` // size in MB
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

func (c *VM) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			// state := r.(*VM)
		}
	}

	return nil
}
